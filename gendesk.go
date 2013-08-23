package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	version_string = "Desktop File Generator v.0.5.3"
)

var (
	// Global flags
	use_color = true
	verbose   = true
)

// Generate the contents for the .desktop file
func createDesktopContents(name string, genericName string, comment string,
	exec string, icon string, useTerminal bool,
	categories []string, mimeTypes []string,
	startupNotify bool) *bytes.Buffer {
	var buf []byte
	b := bytes.NewBuffer(buf)
	b.WriteString("[Desktop Entry]\n")
	b.WriteString("Encoding=UTF-8\n")
	b.WriteString("Type=Application\n")
	b.WriteString("Name=" + name + "\n")
	if genericName != "" {
		b.WriteString("GenericName=" + genericName + "\n")
	}
	b.WriteString("Comment=" + comment + "\n")
	b.WriteString("Exec=" + exec + "\n")
	b.WriteString("Icon=" + icon + "\n")
	if useTerminal {
		b.WriteString("Terminal=true\n")
	} else {
		b.WriteString("Terminal=false\n")
	}
	if startupNotify {
		b.WriteString("StartupNotify=true\n")
	} else {
		b.WriteString("StartupNotify=false\n")
	}
	b.WriteString("Categories=" + strings.Join(categories, ";") + ";\n")
	if len(mimeTypes) > 0 {
		b.WriteString("MimeType=" + strings.Join(mimeTypes, ";") + ";\n")
	}
	return b
}

// Capitalize a string or return the same if it is too short
func capitalize(s string) string {
	if len(s) >= 2 {
		return strings.ToTitle(s[0:1]) + s[1:]
	}
	return s
}

// Write the .desktop file as generated by createDesktopContents
func writeDesktopFile(pkgname string, name string, comment string, exec string,
	useTerminal bool, categories string, genericName string, mimeTypes string, custom string) {
	var categoryList []string
	var mimeTypeList []string

	if len(categories) == 0 {
		categoryList = []string{"Application"}
	} else {
		categoryList = strings.Split(categories, ";")
	}
	// mimeTypeList is an empty []string, or...
	if len(mimeTypes) != 0 {
		mimeTypeList = strings.Split(mimeTypes, ";")
	}

	// mimeTypes may be empty. Disabled terminal
	// and startupnotify for now.
	buf := createDesktopContents(name, genericName, comment, exec, pkgname,
		useTerminal, categoryList, mimeTypeList, false)
	if custom != "" {
		// Write the custom string to the end of the .desktop file (may contain \n)
		buf.WriteString(custom + "\n")
	}
	ioutil.WriteFile(pkgname+".desktop", buf.Bytes(), 0666)
}

// Checks if a trimmed line starts with a specific word
func startsWith(line string, word string) bool {
	//return 0 == strings.Index(strings.TrimSpace(line), word)
	return strings.HasPrefix(line, word)
}

// Return what's between two strings, "a" and "b", in another string
func between(orig string, a string, b string) string {
	if strings.Contains(orig, a) && strings.Contains(orig, b) {
		posa := strings.Index(orig, a) + len(a)
		posb := strings.LastIndex(orig, b)
		return orig[posa:posb]
	}
	return ""
}

// Return the contents between "" or '' (or an empty string)
func betweenQuotes(orig string) string {
	var s string
	for _, quote := range []string{"\"", "'"} {
		s = between(orig, quote, quote)
		if s != "" {
			return s
		}
	}
	return ""
}

// Return the string between the quotes or after the "=", if possible
// or return the original string
func betweenQuotesOrAfterEquals(orig string) string {
	s := betweenQuotes(orig)
	// Check for exactly one "="
	if (s == "") && (strings.Count(orig, "=") == 1) {
		s = strings.TrimSpace(strings.Split(orig, "=")[1])
	}
	return s
}

// Does a keyword exist in the string?
// Disregards several common special characters (like -, _ and .)
func has(s string, kw string) bool {
	lowercase := strings.ToLower(s)
	// Remove the most common special characters
	massaged := strings.Trim(lowercase, "-_.,!?()[]{}\\/:;+@")
	words := strings.Split(massaged, " ")
	for _, word := range words {
		if word == kw {
			return true
		}
	}
	return false
}

// Check if a keyword appears in a package description
func keywordsInDescription(pkgdesc string, keywords []string) bool {
	for _, keyword := range keywords {
		if has(pkgdesc, keyword) {
			return true
		}
	}
	return false
}

func WriteDefaultIconFile(pkgname string, o *Output) error {
	defaultIconFilename := "/usr/share/pixmaps/default.png"
	b, err := ioutil.ReadFile(defaultIconFilename)
	if err != nil {
		o.ErrText("could not read " + defaultIconFilename + "!")
		os.Exit(1)
	}
	filename := pkgname + ".png"
	err = ioutil.WriteFile(filename, b, 0666)
	if err != nil {
		o.ErrText("could not write icon to " + filename + "!")
		os.Exit(1)
	}
	return nil
}

// Use a function for each element in a string list and
// return the modified list
func stringMap(f func(string) string, stringlist []string) []string {
	newlist := make([]string, len(stringlist))
	for i, elem := range stringlist {
		newlist[i] = f(elem)
	}
	return newlist
}

// Return a list of pkgnames for split packages
// or just a list with the pkgname for regular packages
func pkgList(splitpkgname string) []string {
	center := between(splitpkgname, "(", ")")
	if center == "" {
		center = splitpkgname
	}
	if strings.Contains(center, " ") {
		unquoted := strings.Replace(center, "\"", "", -1)
		unquoted = strings.Replace(unquoted, "'", "", -1)
		return strings.Split(unquoted, " ")
	}
	return []string{splitpkgname}
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {

	var filename string
	version_help := "Show application name and version"
	nodownload_help := "Don't download anything"
	nocolor_help := "Don't use colors"
	quiet_help := "Don't output anything on stdout"
	pkgname_help := "The name of the package"
	pkgdesc_help := "Description of the package"
	name_help := "Name of the shortcut"
	genericname_help := "Type of application"
	comment_help := "Shortcut comment"
	exec_help := "Path to executable"
	iconurl_help := "URL to icon"
	terminal_help := "Run the application in a terminal (default is false)"
	categories_help := "Categories, see other .desktop files for examples"
	mimetypes_help := "Mime types, see other .desktop files for examples"
	//startupnotify_help := "Use this is the application takes a year to start and the user needs to know"
	custom_help := "Custom line to append at the end of the .desktop file"

	flag.Usage = func() {
		fmt.Println()
		fmt.Println(version_string)
		fmt.Println("generates .desktop files from a PKGBUILD")
		fmt.Println()
		fmt.Println("Syntax: gendesk [flags] [PKGBUILD filename]")
		fmt.Println()
		fmt.Println("Possible flags:")
		fmt.Println("    --version                    " + version_help)
		fmt.Println("    -n                           " + nodownload_help)
		fmt.Println("    --nocolor                    " + nocolor_help)
		fmt.Println("    -q                           " + quiet_help)
		fmt.Println("    --pkgname=PKGNAME            " + pkgname_help)
		fmt.Println("    --pkgdesc=PKGDESC            " + pkgdesc_help)
		fmt.Println("    --name=NAME                  " + name_help)
		fmt.Println("    --genericname=GENERICNAME    " + genericname_help)
		fmt.Println("    --comment=COMMENT            " + comment_help)
		fmt.Println("    --exec=EXEC                  " + exec_help)
		fmt.Println("    --iconurl=ICON               " + iconurl_help)
		fmt.Println("    --terminal=[true|false]      " + terminal_help)
		fmt.Println("    --categories=CATEGORIES      " + categories_help)
		fmt.Println("    --mimetypes=MIMETYPES        " + mimetypes_help)
		//fmt.Println("    --startupnotify=[true|false] " + startupnotify_help)
		fmt.Println("    --custom=CUSTOM              " + custom_help)
		fmt.Println("    --help                       This text")
		fmt.Println()
		fmt.Println("Note:")
		fmt.Println("    * Either use a PKGBUILD or a bunch of arguments")
		fmt.Println("    * \"../PKGBUILD\" is the default filename")
		fmt.Println("    * _exec in the PKGBUILD can be used to specific a")
		fmt.Println("      different executable for the .desktop file")
		fmt.Println("      Example: _exec=('appname-gui')")
		fmt.Println("    * Split packages are supported")
		fmt.Println("    * If a .png or .svg icon is not found as a file or in")
		fmt.Println("      the PKGBUILD, an icon will be downloaded from:")
		shortname := strings.Split(icon_search_url, "/")
		firstpart := strings.Join(shortname[:3], "/")
		fmt.Println("      " + firstpart)
		fmt.Println("      This may or may not result in the icon you wished for.")
		fmt.Println("    * Categories are guessed based on keywords in the")
		fmt.Println("      package description, but there's also _categories=().")
		fmt.Println("    * Icons are assumed to be installed to")
		fmt.Println("      \"/usr/share/pixmaps/\" by the PKGBUILD")
		fmt.Println()
	}
	version := flag.Bool("version", false, version_help)
	nodownload := flag.Bool("n", false, nodownload_help)
	nocolor := flag.Bool("nocolor", false, nocolor_help)
	quiet := flag.Bool("q", false, quiet_help)
	givenPkgname := flag.String("pkgname", "", pkgname_help)
	givenPkgdesc := flag.String("pkgdesc", "", pkgdesc_help)
	name := flag.String("name", "", name_help)
	genericname := flag.String("genericname", "", genericname_help)
	comment := flag.String("comment", "", comment_help)
	exec := flag.String("exec", "", exec_help)
	givenIconurl := flag.String("iconurl", "", iconurl_help)
	terminal := flag.Bool("terminal", false, terminal_help)
	categories := flag.String("categories", "", categories_help)
	mimetypes := flag.String("mimetypes", "", mimetypes_help)
	custom := flag.String("custom", "", custom_help)
	//startupnotify := flag.Bool("startupnotify", false, startupnotify_help)
	flag.Parse()
	args := flag.Args()

	// New output. Color? Enabled?
	o := NewOutput(!*nocolor, !*quiet)

	if *version {
		o.Println(version_string)
		os.Exit(0)
	}

	pkgname := *givenPkgname
	pkgdesc := *givenPkgdesc
	manualIconurl := *givenIconurl

	if pkgname == "" {
		if len(args) == 0 {
			if os.Getenv("pkgname") == "" {
				filename = "../PKGBUILD"
			} else {
				pkgname = os.Getenv("pkgname")
			}
		} else if len(args) == 1 {
			filename = args[0]
		}
	}

	// Environment variables

	if pkgdesc == "" {
		// $pkgdesc is either empty or not
		pkgdesc = os.Getenv("pkgdesc")
	}
	if *exec == "" {
		*exec = os.Getenv("_exec")
	}
	if *name == "" {
		*name = os.Getenv("_name")
	}
	if *genericname == "" {
		*genericname = os.Getenv("_genericname")
	}
	if *mimetypes == "" {
		*mimetypes = os.Getenv("_mimetypes")
	}
	// support "_mimetype" as well (deprecated)
	if *mimetypes == "" {
		*mimetypes = os.Getenv("_mimetype")
	}
	if *comment == "" {
		*comment = os.Getenv("_comment")
	}
	if *categories == "" {
		*categories = os.Getenv("_categories")
	}
	if *custom == "" {
		*custom = os.Getenv("_custom")
	}

	var pkgnames []string
	var iconurl string

	// Several fields are stored per pkgname, hence the maps
	pkgdescMap := make(map[string]string)
	execMap := make(map[string]string)
	nameMap := make(map[string]string)
	genericNameMap := make(map[string]string)
	mimeTypesMap := make(map[string]string)
	commentMap := make(map[string]string)
	categoriesMap := make(map[string]string)
	customMap := make(map[string]string)

	if filename == "" {
		// Fill in the dictionaries using the arguments
		pkgnames = []string{pkgname}
		if pkgdesc != "" {
			pkgdescMap[pkgname] = pkgdesc
		}
		if *exec != "" {
			execMap[pkgname] = *exec
		}
		if *name != "" {
			nameMap[pkgname] = *name
		}
		if *genericname != "" {
			genericNameMap[pkgname] = *genericname
		}
		if *mimetypes != "" {
			mimeTypesMap[pkgname] = *mimetypes
		}
		if *comment != "" {
			commentMap[pkgname] = *comment
		}
		if *categories != "" {
			categoriesMap[pkgname] = *categories
		}
		if *custom != "" {
			customMap[pkgname] = *custom
		}
	} else {
		// Fill in the dictionaries using a PKGBUILD
		filedata, err := ioutil.ReadFile(filename)
		if err != nil {
			o.ErrText("Could not read " + filename)
			os.Exit(1)
		}
		filetext := string(filedata)
		for _, line := range strings.Split(filetext, "\n") {
			if startsWith(line, "pkgname") {
				pkgname = betweenQuotesOrAfterEquals(line)
				pkgnames = pkgList(pkgname)
				// Select the first pkgname in the array as the "current" pkgname
				if len(pkgnames) > 0 {
					pkgname = pkgnames[0]
				}
			} else if startsWith(line, "package_") {
				pkgname = between(line, "_", "(")
			} else if startsWith(line, "pkgdesc") {
				// Description for the package
				pkgdesc := betweenQuotesOrAfterEquals(line)
				// Use the last found pkgname as the key
				if pkgname != "" {
					pkgdescMap[pkgname] = pkgdesc
				}
			} else if startsWith(line, "_exec") {
				// Custom executable for the .desktop file per (split) package
				exec := betweenQuotesOrAfterEquals(line)
				// Use the last found pkgname as the key
				if pkgname != "" {
					execMap[pkgname] = exec
				}
			} else if startsWith(line, "_name") {
				// Custom Name for the .desktop file per (split) package
				name := betweenQuotesOrAfterEquals(line)
				// Use the last found pkgname as the key
				if pkgname != "" {
					nameMap[pkgname] = name
				}
			} else if startsWith(line, "_genericname") {
				// Custom GenericName for the .desktop file per (split) package
				genericName := betweenQuotesOrAfterEquals(line)
				// Use the last found pkgname as the key
				if (pkgname != "") && (genericName != "") {
					genericNameMap[pkgname] = genericName
				}
			} else if startsWith(line, "_mimetype") {
				// Custom MimeType for the .desktop file per (split) package
				mimeType := betweenQuotesOrAfterEquals(line)
				// Use the last found pkgname as the key
				if pkgname != "" {
					mimeTypesMap[pkgname] = mimeType
				}
			} else if startsWith(line, "_comment") {
				// Custom Comment for the .desktop file per (split) package
				comment := betweenQuotesOrAfterEquals(line)
				// Use the last found pkgname as the key
				if pkgname != "" {
					commentMap[pkgname] = comment
				}
			} else if startsWith(line, "_custom") {
				// Custom string to be added to the end
				// of the .desktop file in question
				custom := betweenQuotesOrAfterEquals(line)
				// Use the last found pkgname as the key
				if pkgname != "" {
					customMap[pkgname] = custom
				}
			} else if startsWith(line, "_categories") {
				categories := betweenQuotesOrAfterEquals(line)
				// Use the last found pkgname as the key
				if pkgname != "" {
					categoriesMap[pkgname] = categories
				}
			} else if strings.Contains(line, "http://") &&
				strings.Contains(line, ".png") {
				// Only supports png icons downloaded over http,
				// picks the first fitting url
				if iconurl == "" {
					iconurl = "h" + between(line, "h", "g") + "g"
					if strings.Contains(iconurl, "$pkgname") {
						iconurl = strings.Replace(iconurl,
							"$pkgname", pkgname, -1)
					}
					if strings.Contains(iconurl, "${pkgname}") {
						iconurl = strings.Replace(iconurl,
							"${pkgname}", pkgname, -1)
					}
					if strings.Contains(iconurl, "$") {
						// If there are more $variables, don't bother (for now)
						// TODO: replace all defined $variables...
						iconurl = ""
					}
				}
			}
		}
	}

	// Write .desktop and .png icon for each package
	for _, pkgname := range pkgnames {
		if strings.Contains(pkgname, "-nox") ||
			strings.Contains(pkgname, "-cli") {
			// Don't bother if it's a -nox or -cli package
			continue
		}
		pkgdesc, found := pkgdescMap[pkgname]
		if !found {
			// Fall back on the package name
			pkgdesc = pkgname
		}
		exec, found := execMap[pkgname]
		if !found {
			// Fall back on the package name
			exec = pkgname
		}
		name, found := nameMap[pkgname]
		if !found {
			// Fall back on the capitalized package name
			name = capitalize(pkgname)
		}
		genericName, found := genericNameMap[pkgname]
		if !found {
			// Fall back on no generic name
			genericName = ""
		}
		comment, found := commentMap[pkgname]
		if !found {
			// Fall back on pkgdesc
			comment = pkgdesc
		}
		mimeTypes, found := mimeTypesMap[pkgname]
		if !found {
			// Fall back on no mime type
			mimeTypes = ""
		}
		custom, found := customMap[pkgname]
		if !found {
			// Fall back on no custom additional lines
			custom = ""
		}
		categories, found := categoriesMap[pkgname]
		if !found {
			categories = GuessCategory(pkgdesc)
		}
		const nSpaces = 32
		spaces := strings.Repeat(" ", nSpaces)[:nSpaces-min(nSpaces, len(pkgname))]
		if o.IsEnabled() {
			fmt.Printf("%s%s%s%s%s ",
				o.DarkGrayText("["), o.LightBlueText(pkgname),
				o.DarkGrayText("]"), spaces,
				o.DarkGrayText("Generating desktop file..."))
		}
		writeDesktopFile(pkgname, name, comment, exec,
			*terminal, categories, genericName, mimeTypes, custom)
		if o.IsEnabled() {
			fmt.Printf("%s\n", o.DarkGreenText("ok"))
		}

		// Download an icon if it's not downloaded by
		// the PKGBUILD and not there already (.png or .svg)
		pngFilenames, _ := filepath.Glob("*.png")
		svgFilenames, _ := filepath.Glob("*.svg")
		if ((0 == (len(pngFilenames) + len(svgFilenames))) && (iconurl == "")) && (*nodownload == false) {
			if len(pkgname) < 1 {
				o.ErrText("No pkgname, can't download icon")
				os.Exit(1)
			}
			fmt.Printf("%s%s%s%s%s ",
				o.DarkGrayText("["), o.LightBlueText(pkgname),
				o.DarkGrayText("]"), spaces,
				o.DarkGrayText("Downloading icon..."))
			var err error
			if manualIconurl == "" {
				err = WriteIconFile(pkgname, o)
			} else {
				// Default filename
				iconFilename := pkgname + ".png"
				// Get the last part of the URL, after the "/" to use as the filename
				if strings.Contains(manualIconurl, "/") {
					pos := strings.LastIndex(manualIconurl, "/")
					iconFilename = manualIconurl[pos+1:]
				}
				err = DownloadFile(manualIconurl, iconFilename, o)
			}
			if err == nil {
				if o.IsEnabled() {
					fmt.Printf("%s\n", o.LightCyanText("ok"))
				}
			} else {
				if o.IsEnabled() {
					fmt.Printf("%s\n", o.DarkYellowText("no"))
					fmt.Printf("%s%s%s%s%s ",
						o.DarkGrayText("["),
						o.LightBlueText(pkgname),
						o.DarkGrayText("]"),
						spaces,
						o.DarkGrayText("Using default icon instead..."))
				}
				err := WriteDefaultIconFile(pkgname, o)
				if err == nil {
					if o.IsEnabled() {
						fmt.Printf("%s\n", o.LightPurpleText("yes"))
					}
				}
			}
		}
	}
}
