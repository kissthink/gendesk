package main

import (
	"bytes"
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"hash"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	version_string  = "Desktop File Generator v.0.3.1"
	icon_search_url = "https://admin.fedoraproject.org/pkgdb/appicon/show/%s"
)

var (
	multimedia_kw = []string{"video", "audio", "sound", "graphics", "draw", "demo"}
	network_kw    = []string{"network", "p2p"}
	audiovideo_kw = []string{"synth", "synthesizer"}
	editor_kw     = []string{"editor"}
	science_kw    = []string{"gps", "inspecting"}
	vcs_kw        = []string{"git"}
	// Emulator and player aren't always for games, but those cases should be picked up by one of the other categories first
	game_kw          = []string{"game", "rts", "mmorpg", "emulator", "player"}
	arcadegame_kw    = []string{"combat", "arcade", "racing", "fighting", "fight"}
	actiongame_kw    = []string{"shooter", "fps"}
	adventuregame_kw = []string{"roguelike", "rpg"}
	logicgame_kw     = []string{"puzzle"}
	programming_kw   = []string{"code", "c", "ide", "programming", "develop", "compile"}

	// Global flags
	use_color = true
	verbose   = true
)

// Generate the contents for the .desktop file
func createDesktopContents(name string, genericName string, comment string, exec string, icon string, useTerminal bool, categories []string, mimeTypes []string, startupNotify bool) *bytes.Buffer {
	var buf []byte
	b := bytes.NewBuffer(buf)
	b.WriteString("[Desktop Entry]\n")
	b.WriteString("Encoding=UTF-8\n")
	b.WriteString("Type=Application\n")
	b.WriteString("Name=" + name + "\n")
	b.WriteString("GenericName=" + genericName + "\n")
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
	b.WriteString("Type=Application\n")
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
func writeDesktopFile(pkgname string, name string, comment string, exec string, categories string, genericName string, mimeTypes string) {
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

	// Only supports png icons, mimeTypes may be empty. Disabled terminal and startupnotify for now.
	buf := createDesktopContents(name, genericName, comment, exec, pkgname+".png", false, categoryList, mimeTypeList, false)
	ioutil.WriteFile(pkgname+".desktop", buf.Bytes(), 0666)
}

func startsWith(line string, word string) bool {
	return 0 == strings.Index(strings.TrimSpace(line), word)
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

func betweenQuotesOrAfterEquals(orig string) string {
	s := betweenQuotes(orig)
	// If the string is not between quotes, get the text after "="
	if (s == "") && (strings.Count(orig, "=") == 1) {
		s = strings.TrimSpace(strings.Split(orig, "=")[1])
	}
	return s
}

// TODO: Improve the keyword check algorithm to be able to check for the keyword "C" properly
// Does a keyword exist in a lowercase string?
func has(s string, kw string) bool {
	// Replace "-" with " " when searching for keywords.
	// Checking for " " + kw can definitely be improved.
	return -1 != strings.Index(strings.Replace(strings.ToLower(s), "-", " ", -1), kw+" ")
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

// Download icon from the search url in icon_search_url
func writeIconFile(pkgname string) error {
	// Only supports png icons
	filename := pkgname + ".png"
	var client http.Client
	resp, err := client.Get(fmt.Sprintf(icon_search_url, capitalize(pkgname)))
	if err != nil {
		if verbose {
			fmt.Println(darkRedText("Could not download icon"))
		}
		os.Exit(1)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if verbose {
			fmt.Println(darkRedText("Could not dump body"))
		}
		os.Exit(1)
	}

	var h hash.Hash = md5.New()
	h.Write(b)
	//fmt.Printf("Icon MD5: %x\n", h.Sum())

	// If the icon is the "No icon found" icon (known hash), return with an error
	if fmt.Sprintf("%x", h.Sum(nil)) == "12928aa3233965175ea30f5acae593bf" {
		return errors.New("No icon found")
	}

	err = ioutil.WriteFile(filename, b, 0666)
	if err != nil {
		if verbose {
			fmt.Println(darkRedText("Could not write icon to " + filename + "!"))
		}
		os.Exit(1)
	}
	return nil
}

func writeDefaultIconFile(pkgname string) error {
	defaultIconFilename := "/usr/share/pixmaps/default.png"
	b, err := ioutil.ReadFile(defaultIconFilename)
	if err != nil {
		if verbose {
			fmt.Println(darkRedText("could not read " + defaultIconFilename + "!"))
		}
		os.Exit(1)
	}
	filename := pkgname + ".png"
	err = ioutil.WriteFile(filename, b, 0666)
	if err != nil {
		if verbose {
			fmt.Println(darkRedText("could not write icon to " + filename + "!"))
		}
		os.Exit(1)
	}
	return nil
}

// Download a file
func downloadFile(url string, filename string) {
	var client http.Client
	resp, err := client.Get(url)
	if err != nil {
		if verbose {
			fmt.Println(darkRedText("Could not download file"))
		}
		os.Exit(1)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if verbose {
			fmt.Println(darkRedText("Could not dump body"))
		}
		os.Exit(1)
	}
	err = ioutil.WriteFile(filename, b, 0666)
	if err != nil {
		if verbose {
			fmt.Println(darkRedText("Could not write data to " + filename + "!"))
		}
		os.Exit(1)
	}
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
		unquoted = strings.Replace(center, "'", "", -1)
		return strings.Split(unquoted, " ")
	}
	return []string{splitpkgname}
}

func colorOn(num1 int, num2 int) string {
	if use_color {
		return fmt.Sprintf("\033[%d;%dm", num1, num2)
	}
	return ""
}

func colorOff() string {
	if use_color {
		return "\033[0m"
	}
	return ""
}

func darkRedText(s string) string {
	return colorOn(0, 31) + s + colorOff()
}

func lightGreenText(s string) string {
	return colorOn(1, 32) + s + colorOff()
}

func darkGreenText(s string) string {
	return colorOn(0, 32) + s + colorOff()
}

func lightYellowText(s string) string {
	return colorOn(1, 33) + s + colorOff()
}

func darkYellowText(s string) string {
	return colorOn(0, 33) + s + colorOff()
}

func lightBlueText(s string) string {
	return colorOn(1, 34) + s + colorOff()
}

func darkBlueText(s string) string {
	return colorOn(0, 34) + s + colorOff()
}

func lightCyanText(s string) string {
	return colorOn(1, 36) + s + colorOff()
}

func lightPurpleText(s string) string {
	return colorOn(1, 35) + s + colorOff()
}

func darkPurpleText(s string) string {
	return colorOn(0, 35) + s + colorOff()
}

func darkGrayText(s string) string {
	return colorOn(1, 30) + s + colorOff()
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
	flag.Usage = func() {
		fmt.Println()
		fmt.Println(version_string)
		fmt.Println("generates .desktop files from a PKGBUILD")
		fmt.Println()
		fmt.Println("Syntax: gendesk [flags] filename")
		fmt.Println()
		fmt.Println("Possible flags:")
		fmt.Println("    * --version        " + version_help)
		fmt.Println("    * -n               " + nodownload_help)
		fmt.Println("    * --nocolor        " + nocolor_help)
		fmt.Println("    * -q               " + quiet_help)
		fmt.Println("    * --help           This text")
		fmt.Println()
		fmt.Println("Note:")
		fmt.Println("    * \"../PKGBUILD\" is the default filename")
		fmt.Println("    * _exec in the PKGBUILD can be used to specific a different executable for the .desktop file")
		fmt.Println("      Example: _exec=('appname-gui')")
		fmt.Println("    * Split packages are supported")
		fmt.Println("    * If a .png icon is not found as a file or in the PKGBUILD, an icon will be downloaded from:")
		fmt.Println("      " + icon_search_url)
		fmt.Println("      This may or may not result in the icon you wished for.")
		fmt.Println("    * Categories are guessed based on keywords in the package description")
		fmt.Println("    * Icons are assumed to be installed to \"/usr/share/pixmaps/$pkgname.png\" by the PKGBUILD")
		fmt.Println()
	}
	version := flag.Bool("version", false, version_help)
	nodownload := flag.Bool("n", false, nodownload_help)
	nocolor := flag.Bool("nocolor", false, nocolor_help)
	quiet := flag.Bool("q", false, quiet_help)
	flag.Parse()
	args := flag.Args()
	if *version {
		fmt.Println(version_string)
		os.Exit(0)
	} else if len(args) == 0 {
		filename = "../PKGBUILD"
	} else if len(args) == 1 {
		filename = args[0]
	} else {
		fmt.Println(darkRedText("Too many arguments"))
		os.Exit(1)
	}

	use_color = !*nocolor
	verbose = !*quiet

	filedata, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, darkRedText("Could not read %s\n"), filename)
		os.Exit(1)
	}
	filetext := string(filedata)

	var pkgname string
	var pkgnames []string
	var iconurl string
	pkgdescMap := make(map[string]string)
	execMap := make(map[string]string)
	nameMap := make(map[string]string)
	genericNameMap := make(map[string]string)
	mimeTypeMap := make(map[string]string)
	commentMap := make(map[string]string)

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
			if pkgname != "" {
				genericNameMap[pkgname] = genericName
			}
		} else if startsWith(line, "_mimetype") {
			// Custom MimeType for the .desktop file per (split) package
			mimeType := betweenQuotesOrAfterEquals(line)
			// Use the last found pkgname as the key
			if pkgname != "" {
				nameMap[pkgname] = mimeType
			}
		} else if startsWith(line, "_comment") {
			// Custom Comment for the .desktop file per (split) package
			comment := betweenQuotesOrAfterEquals(line)
			// Use the last found pkgname as the key
			if pkgname != "" {
				nameMap[pkgname] = comment
			}
		} else if strings.Contains(line, "http://") && strings.Contains(line, ".png") {
			// Only supports png icons downloaded over http, picks the first fitting url
			if iconurl == "" {
				iconurl = "h" + between(line, "h", "g") + "g"
				if strings.Contains(iconurl, "$pkgname") {
					iconurl = strings.Replace(iconurl, "$pkgname", pkgname, -1)
				}
				if strings.Contains(iconurl, "${pkgname}") {
					iconurl = strings.Replace(iconurl, "${pkgname}", pkgname, -1)
				}
				if strings.Contains(iconurl, "$") {
					// Will only replace pkgname. There are more replacements
					iconurl = ""
				}
			}
		}
	}

	//fmt.Println("pkgnames:", pkgnames)

	// Write .desktop and .png icon for each package
	for _, pkgname := range pkgnames {
		if strings.Contains(pkgname, "-nox") || strings.Contains(pkgname, "-cli") {
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
			// Fall back on the package Name
			name = capitalize(name)
		}
		comment, found := commentMap[pkgname]
		if !found {
			// Fall back on pkgdesc
			comment = pkgdesc
		}
		mimeType, found := mimeTypeMap[pkgname]
		if !found {
			// Fall back on no mime type
			mimeType = ""
		}
		// Approximately identify various categories
		categories := ""
		if keywordsInDescription(pkgdesc, multimedia_kw) {
			categories = "Application;Multimedia"
		} else if keywordsInDescription(pkgdesc, network_kw) {
			categories = "Application;Network"
		} else if keywordsInDescription(pkgdesc, audiovideo_kw) {
			categories = "Application;AudioVideo"
		} else if keywordsInDescription(pkgdesc, editor_kw) {
			categories = "Application;Development;TextEditor"
		} else if keywordsInDescription(pkgdesc, science_kw) {
			categories = "Application;Science"
		} else if keywordsInDescription(pkgdesc, vcs_kw) {
			categories = "Application;Development;RevisionControl"
		} else if keywordsInDescription(pkgdesc, arcadegame_kw) {
			categories = "Application;Game;ArcadeGame"
		} else if keywordsInDescription(pkgdesc, actiongame_kw) {
			categories = "Application;Game;ActionGame"
		} else if keywordsInDescription(pkgdesc, adventuregame_kw) {
			categories = "Application;Game;AdventureGame"
		} else if keywordsInDescription(pkgdesc, game_kw) {
			categories = "Application;Game"
		} else if keywordsInDescription(pkgdesc, programming_kw) {
			categories = "Application;Development"
		}
		const nSpaces = 32
		spaces := strings.Repeat(" ", nSpaces)[:nSpaces-min(nSpaces, len(pkgname))]
		if verbose {
			fmt.Printf("%s%s%s%s%s ", darkGrayText("["), lightBlueText(pkgname), darkGrayText("]"), spaces, darkGrayText("Generating desktop file..."))
		}
		writeDesktopFile(pkgname, name, comment, exec, categories, genericName, mimeType)
		if verbose {
			fmt.Printf("%s\n", darkGreenText("ok"))
		}

		// Download an icon if it's not downloaded by the PKGBUILD and not there already
		files, _ := filepath.Glob("*.png")
		if ((len(files) == 0) && (iconurl == "")) && (*nodownload == false) {
			if len(pkgname) < 1 {
				if verbose {
					fmt.Println(darkRedText("No pkgname, can't download icon"))
				}
				os.Exit(1)
			}
			fmt.Printf("%s%s%s%s%s ", darkGrayText("["), lightBlueText(pkgname), darkGrayText("]"), spaces, darkGrayText("Downloading icon..."))
			err := writeIconFile(pkgname)
			if err == nil {
				if verbose {
					fmt.Printf("%s\n", lightCyanText("ok"))
				}
			} else {
				if verbose {
					fmt.Printf("%s\n", darkYellowText("no"))
					fmt.Printf("%s%s%s%s%s ", darkGrayText("["), lightBlueText(pkgname), darkGrayText("]"), spaces, darkGrayText("Using default icon instead..."))
				}
				err := writeDefaultIconFile(pkgname)
				if err == nil {
					if verbose {
						fmt.Printf("%s\n", lightPurpleText("yes"))
					}
				}
			}
		}
	}
}
