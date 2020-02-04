package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	logFile     = "/var/log/pacman.log"
	COLOR_NONE  = "\033[0m"
	COLOR_BLUE  = "\033[0;34m"
	COLOR_GREEN = "\033[0;36m"
	COLOR_RED   = "\033[38;5;124m"
	COLOR_OR    = "\033[38;5;214m"
	COLOR_GRAY  = "\033[38;5;243m"
)

var GitBranch string
var Version string
var BuildDate string
var GitID string

type fn func(string, string, *regexp.Regexp, *int, *[2]string) string

func main() {

	logFileName := logFile
	verArg := flag.Bool("v", false, "Version")
	clearArg := flag.Bool("c", false, "Clear log")
	getCalendar := flag.Bool("a", false, "get calendar")
	pkgArg := flag.String("p", "", "Package to find")
	dateArg := flag.String("d", "", "Date filter (YYYY-MM-DD)")
	fileArg := flag.String("f", "", "Pacman log file")
	flag.Parse()

	if *verArg {
		fmt.Printf("\n%s Version: %v %v %v %v\n", filepath.Base(os.Args[0]), Version, GitID, GitBranch, BuildDate)
	}

	if *fileArg != "" {
		logFileName = *fileArg
	}

	if *clearArg {
		defer os.Remove("/tmp/pacman.log")
		fmt.Println("::pacman-log-clear\n ")
		CopyFile(logFileName, "/tmp/pacman.log")
		convertFile("/tmp/pacman.log", logFileName)
	}

	if len(os.Args) > 1 && *pkgArg == "" && len(flag.Args()) > 0 {
		*pkgArg = flag.Args()[0]
	}

	if *getCalendar {
		cal := CalendarFile(logFileName)

		max := 0
		for _, v := range cal {
			if v > max {
				max = v
			}
		}
		var keys []string
		for k := range cal {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		//for key, value := range cal {
		for _, k := range keys {
			pourcent := (float64(cal[k]) * float64(100)) / float64(max)
			//fmt.Print(k, COLOR_BLUE, cal[k], COLOR_NONE, pourcent) //, math.Round(pourcent))
			fmt.Printf("%s \033[0;34m%4d\033[0m ", k, cal[k] /*, pourcent*/)
			//for i := range int64(math.Round(math.Round(pourcent))) {
			for i := 1; i <= int(math.Round(math.Round(pourcent))/1.5); i++ {
				print("█")
			}
			println("")
			//fmt.Printf(" %4.2f \n", pourcent)
		}
		fmt.Println("\nmax:", max, "packages in a day")
	}

	if *pkgArg != "" {
		fmt.Println("::pacman-log find:", COLOR_GREEN+strings.ToLower(*pkgArg), COLOR_NONE, "\n ")
		parseFile(logFileName, pkgfilter, strings.ToLower(*pkgArg))
	}

	if *dateArg != "" {
		fmt.Println("::pacman-log find date:", COLOR_GREEN+strings.ToLower(*dateArg), COLOR_NONE, "\n ")
		parseFile(logFileName, datefilter, strings.ToLower(*dateArg))
	}

	if len(os.Args) < 2 {
		parseFile(logFileName, allfilter, "")
	}
}

func allfilter(line, filter string, re *regexp.Regexp, transactions *int, dates *[2]string) string {
	if line == "" {
		return ""
	}
	if strings.Index(line, "[PACMAN] synchronizing") > 0 || strings.Index(line, "[PACMAN] starting") > 0 {
		return ""
	}
	if strings.Index(line, "[ALPM] running") > 0 {
		// .hook
		return ""
	}
	if strings.Index(line, "ALPM-SCRIPTLET") < 0 {
		match := re.FindStringSubmatch(line)
		if len(match) < 2 {
			return ""
		}
		if strings.HasSuffix(line, "[ALPM] transaction completed") {
			println("")
			*transactions++
			return ""
		}
		if match[3] == "transaction started" {
			t := strings.Replace(match[1], "T", " ", 1)[:16]
			line = "" + COLOR_BLUE + t + COLOR_NONE + " -> "
			if dates[0] == "" {
				dates[0] = t[:10]
			}
			dates[1] = t[:10]
		} else {
			if match[2] == "ALPM" {
				match[3] = strings.Replace(match[3], "warning:", COLOR_OR+"warning:"+COLOR_NONE, 2)
				line = "  " + COLOR_GRAY + match[2] + ": " + COLOR_NONE + match[3]
			} else {
				// command pacman
				line = "" + match[2] + ":: " + COLOR_GREEN + match[3] + COLOR_NONE
			}

		}
		return line
	}
	return ""
}

func datefilter(line, filter string, re *regexp.Regexp, transactions *int, dates *[2]string) string {
	if line == "" {
		return ""
	}
	if strings.Index(line, filter) < 0 {
		return ""
	}
	if strings.Index(line, "[PACMAN] synchronizing") > 0 || strings.Index(line, "[PACMAN] starting") > 0 {
		return ""
	}
	if strings.Index(line, "[ALPM] running") > 0 {
		// .hook
		return ""
	}
	if strings.Index(line, "ALPM-SCRIPTLET") < 0 {
		match := re.FindStringSubmatch(line)
		if len(match) < 2 {
			return ""
		}
		if strings.HasSuffix(line, "[ALPM] transaction completed") {
			println("")
			*transactions++
			return ""
		}
		if match[3] == "transaction started" {
			t := strings.Replace(match[1], "T", " ", 1)[:16]
			line = "" + COLOR_BLUE + t + COLOR_NONE + " -> "
			if dates[0] == "" {
				dates[0] = t[:10]
			}
			dates[1] = t[:10]
		} else {
			if match[2] == "ALPM" {
				match[3] = strings.Replace(match[3], "warning:", COLOR_OR+"warning:"+COLOR_NONE, 2)
				line = "  " + COLOR_GRAY + match[2] + ": " + COLOR_NONE + match[3]
			} else {
				// command pacman
				line = "" + match[2] + ":: " + COLOR_GREEN + match[3] + COLOR_NONE
			}

		}
		return line
	}
	return ""
}

func pkgfilter(line, filter string, re *regexp.Regexp, transactions *int, dates *[2]string) string {
	if line == "" {
		return ""
	}
	if strings.Index(line, "[PACMAN] synchronizing") > 0 || strings.Index(line, "[PACMAN] starting") > 0 {
		return ""
	}
	if strings.Index(line, "[ALPM] running") > 0 {
		// .hook
		return ""
	}
	if strings.Index(line, " "+filter+" ") < 0 {
		return ""
	}
	if strings.Index(line, "ALPM-SCRIPTLET") < 0 {

		match := re.FindStringSubmatch(line)
		if len(match) < 2 || match[2] != "ALPM" {
			return ""
		}
		*transactions++

		t := strings.Replace(match[1], "T", " ", 1)[:16]
		if dates[0] == "" {
			dates[0] = t[:10]
		}
		c := "+"
		if strings.Index(match[3], "installed") == 0 {
			c = "*"
			match[3] = strings.Replace(match[3], "installed", "installed  ", 1)
		}
		if strings.Index(match[3], "reinstalled") == 0 {
			c = "="
		}
		if strings.Index(match[3], "downgraded") == 0 {
			c = "-"
			match[3] = strings.Replace(match[3], "downgraded", "downgraded ", 1)
		}
		if strings.Index(match[3], "removed") == 0 {
			c = "."
			match[3] = strings.Replace(match[3], "removed", "removed    ", 1)
		}

		match[3] = strings.Replace(match[3], "upgraded", "upgraded   ", 1)
		dates[1] = t[:10]
		match[3] = strings.Replace(match[3], filter, COLOR_GREEN+filter+COLOR_NONE, 2)
		return fmt.Sprint(COLOR_BLUE, t, COLOR_NONE, "  ",
			COLOR_GRAY, c+COLOR_NONE, " ",
			match[3])
	}
	return ""
}

func parseFile(logFile string, fn fn, strFilter string) {
	transactions := 0
	var dates [2]string
	re := regexp.MustCompile(`.(?P<dat>.*)\]\s+\[(?P<verb>\S+)\]\s+(?P<txt>.*)`)

	file, err := os.Open(logFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = fn(line, strFilter, re, &transactions, &dates)
		if line != "" {
			fmt.Println(line)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("\n::", transactions, "transactions  ", dates[0], " -> ", dates[1])
}

func CalendarFile(logFile string) (calendar map[string]int) {
	//re := regexp.MustCompile(`.(?P<dat>.*)\]\s+\[(?P<verb>\S+)\]\s+(?P<txt>.*)`)
	calendar = make(map[string]int)
	now := time.Now().AddDate(0, -2, 0)
	//calendar["2020-01-15"] = 0
	//calendar["2020-01-25"] = 5

	file, err := os.Open(logFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Index(line, "[ALPM] upgraded") > 0 ||
			strings.Index(line, "[ALPM] reinstalled") > 0 ||
			strings.Index(line, "[ALPM] downgraded") > 0 ||
			strings.Index(line, "[ALPM] removed") > 0 ||
			strings.Index(line, "[ALPM] installed") > 0 {
			d := line[1:11]
			if _, ok := calendar[d]; ok {
				calendar[d]++
			} else {
				t, _ := time.Parse("2006-01-02", d)
				if t.After(now) {
					calendar[d] = 1
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return calendar
}

func convertFile(rlog, wlog string) {

	transactions := 0

	file, err := os.Open(rlog)
	if err != nil {
		fmt.Println(err)
		os.Exit(127)
	}
	defer file.Close()

	os.Remove(wlog)
	fileOut, err := os.OpenFile(wlog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("failed to write file:", err)
		os.Exit(126)
	}
	defer fileOut.Close()
	datawriter := bufio.NewWriter(fileOut)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		/*
		   TODO
		   garder les warning et errors ...
		   - conflit de fichier
		   - chmod ...
		   - pacnew
		*/
		if line == "" {
			println("--------- vide...")
			continue
		}
		if strings.Index(line, "[PACMAN] synchronizing") > 0 || strings.Index(line, "[PACMAN] starting") > 0 {
			continue
		}
		if strings.Index(line, "[ALPM] running") > 0 {
			// .hook
			continue
		}
		if strings.Index(line, "ALPM-SCRIPTLET") < 0 {
			if strings.HasSuffix(line, "[ALPM] transaction completed") {
				line = line + "\n"
				transactions++
			}
			if strings.HasSuffix(line, "[ALPM] transaction started") {
				line = "\n" + line
			}
			fmt.Println(line)
			if _, err = datawriter.WriteString(line + "\n"); err != nil {
				log.Fatalf("failed write in file: %s", err)
			}
		}
	}
	datawriter.Flush()

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n::", transactions, "transactions")
}

func CopyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}
