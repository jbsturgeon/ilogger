package ilogger

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	yaml "gopkg.in/yaml.v2"
)

const (
	whiteEnum = (1 << iota)
	cyanEnum
	blueEnum
	greenEnum
	yellowEnum
	redEnum
	magentaEnum
)

var (
	colorMap = map[LogLevel]int{}
)

const (
	logLevelEnv    = "LOG_LEVEL"
	colorConfigEnv = "LOG_COLOR_CONFIG"

	debugPrefix = "DEBUG - "
	infoPrefix  = "INFO - "
	warnPrefix  = "WARN - "
	errorPrefix = "ERROR - "
)

// LogColor type used to specify log level and color
type LogColor struct {
	Level string
	Color string
}

var (
	logLevelConfig = os.Getenv(logLevelEnv)
	showColors     bool
)

// Logging levels
const (
	LMandatory = LogLevel(1 << iota)
	LError
	LWarn
	LInfo
	LDebug
)

// LogLevel is a logging level
type LogLevel uint8

// ILog struct for logging variables
type ILog struct {
	Path  string
	Level LogLevel

	fileDay int
	logFile *os.File
	logOpen bool
	iLog    *log.Logger
}

func init() {
	// setup colorMap
	colorConfig := os.Getenv(colorConfigEnv)
	if colorConfig != "" {
		colors, err := ioutil.ReadFile(colorConfig)
		if err != nil {
			fmt.Printf("Unable to get colors from color config file, Error: %+v\n", err)
		} else {
			var colorList []LogColor
			if err = yaml.Unmarshal(colors, &colorList); err != nil {
				fmt.Printf("Unable to unmarshal colors from config file, Error: %+v\n", err)
			} else {
				showColors = true
				for _, c := range colorList {
					prefixEnum, colorEnum := mapColor(c.Level, c.Color)
					colorMap[prefixEnum] = colorEnum
				}
			}
		}
	}
}

func mapColor(prefix, colorChoice string) (LogLevel, int) {
	var prefixEnum LogLevel
	var colorEnum int

	switch strings.ToUpper(prefix) {
	case "DEBUG":
		prefixEnum = LDebug
	case "INFO":
		prefixEnum = LInfo
	case "WARN":
		prefixEnum = LWarn
	case "ERROR":
		prefixEnum = LError
	default:
		prefixEnum = LogLevel(0)
	}

	switch strings.ToUpper(colorChoice) {
	case "WHITE":
		colorEnum = whiteEnum
	case "BLUE":
		colorEnum = blueEnum
	case "CYAN":
		colorEnum = cyanEnum
	case "GREEN":
		colorEnum = greenEnum
	case "YELLOW":
		colorEnum = yellowEnum
	case "RED":
		colorEnum = redEnum
	case "MAGENTA":
		colorEnum = magentaEnum
	default:
		colorEnum = -1
	}

	return prefixEnum, colorEnum
}

// NewFile attaches a new file for the instance logger to write to
func (i *ILog) NewFile(p string, d, l int) error {
	// validate input
	if len(p) == 0 {
		log.Fatalf("ILog filepath not set: %v", "zero length")
	}

	i.Path = p
	i.fileDay = d

	// validate directory
	if err := os.MkdirAll(i.Path, 0755); err != nil {
		log.Fatalf("Cannot make log path (%v): %v", i.Path, err)
	}

	// validate / close current file
	if i.logOpen {
		if err := i.logFile.Close(); err != nil {
			log.Printf("unable to close logger (%s): %+v", i.logFile.Name(), err)
		}
	}

	//set LogLevel
	if l < 0 {
		i.SetLogLevel(logLevelConfig)
	} else {
		i.Level = LogLevel(l)
	}

	t := time.Now().UTC()

	ex, err := os.Executable()
	bex := filepath.Base(ex)
	name := fmt.Sprintf("%si_%s_%s_%s.log", bex, t.Format("2006"), t.Format("01"), t.Format("02"))
	name = filepath.Join(i.Path, name)

	i.logFile, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("unable to open logger (%s): %+v", i.logFile.Name(), err)
	}

	//setup golang log variable; we could default to os.Stderr or os.Stdout???
	i.iLog = log.New(i.logFile, "", log.LstdFlags|log.Lshortfile)

	i.logOpen = true
	i.fileDay = t.Day()

	return nil
}

// SetLogLevel allows applications to change the log level with a reload instead of restart
func (i *ILog) SetLogLevel(level string) {
	switch strings.ToUpper(level) {
	case "ERROR":
		i.Level = LError
	case "WARN":
		i.Level = LWarn
	case "INFO":
		i.Level = LInfo
	case "DEBUG":
		i.Level = LDebug
	default:
		i.Level = LError
	}
}

// Log sends the format and the params to the underlying logger
func (i *ILog) Log(level LogLevel, formattedString string, params ...interface{}) {
	if level > i.Level {
		return
	}

	curTime := time.Now().UTC()
	curDay := curTime.Day()

	if i != nil && (!i.logOpen || curDay != i.fileDay) {
		if err := i.NewFile(i.Path, curDay, int(i.Level)); err != nil {
			log.Fatalf("Unable to create new ILog: %v", "zero length")
		}
	}

	if _, err := os.Stat(i.logFile.Name()); err != nil {
		if err := i.NewFile(i.Path, curDay, int(i.Level)); err != nil {
			log.Fatalf("Unable to create ILog: %v", "zero length")
		}
	}

	// log message
	i.iLog.Output(3, i.paintString(fmt.Sprintf(formattedString, params...), colorMap[level]))
}

// Fatalf is equivalent to calling Errorf followed by os.Exit(1)
func (i *ILog) Fatalf(formattedString string, params ...interface{}) {
	i.Log(LError, formattedString, params...)
	os.Exit(1)
}

// Panic is equivalent to calling Errorf followed by panic(params)
func (i *ILog) Panic(formattedString string, params ...interface{}) {
	s := fmt.Sprintf(formattedString, params...)
	i.Log(LError, formattedString, params...)
	panic(s)
}

// Error log
func (i *ILog) Error(err error) {
	i.Log(LError, err.Error())
}

// Mandatory always logs regardless of logging level
func (i *ILog) Mandatory(formattedString string, params ...interface{}) {
	i.Log(LMandatory, formattedString, params...)
}

// Errorf log
func (i *ILog) Errorf(formattedString string, params ...interface{}) {
	i.Log(LError, errorPrefix+formattedString, params...)
}

// Warn log
func (i *ILog) Warn(formattedString string, params ...interface{}) {
	i.Log(LWarn, warnPrefix+formattedString, params...)
}

// Info log
func (i *ILog) Info(formattedString string, params ...interface{}) {
	i.Log(LInfo, infoPrefix+formattedString, params...)
}

// Debug log
func (i *ILog) Debug(formattedString string, params ...interface{}) {
	i.Log(LDebug, debugPrefix+formattedString, params...)
}

func (i *ILog) paintString(str string, colorEnum int) string {
	if showColors {
		return str
	}

	switch colorEnum {
	case whiteEnum:
		str = color.WhiteString(str)
	case blueEnum:
		str = color.BlueString(str)
	case cyanEnum:
		str = color.CyanString(str)
	case greenEnum:
		str = color.GreenString(str)
	case yellowEnum:
		str = color.YellowString(str)
	case redEnum:
		str = color.RedString(str)
	case magentaEnum:
		str = color.MagentaString(str)
	}

	return str
}
