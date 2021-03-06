// Package logger name declaration
package logger

// Import packages
import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

var (
	// Map for the various codes of colors
	colors map[LogLevel]string

	// Map from format's placeholders to printf verbs
	phfs map[string]string

	// Contains color strings for stdout
	logNo uint64

	// Default format of log message
	defFmt = "#%[1]d %[2]s %[4]s:%[5]d ▶ %.3[6]s %[7]s"

	// Default format of time
	defTimeFmt = "2006-01-02 15:04:05.000"
)

// LogLevel type
type LogLevel int

// Color numbers for stdout
const (
	Black = (iota + 90)
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

// Log Level
const (
	CriticalLevel LogLevel = iota + 1
	ErrorLevel
	WarningLevel
	NoticeLevel
	InfoLevel
	DebugLevel
)

// Worker class, Worker is a log object used to log messages and Color specifies
// if colored output is to be produced
type Worker struct {
	Minion     *log.Logger
	Color      int
	format     string
	timeFormat string
	level      LogLevel
}

// Info class, Contains all the info on what has to logged, time is the current time, Module is the specific module
// For which we are logging, level is the state, importance and type of message logged,
// Message contains the string to be logged, format is the format of string to be passed to sprintf
type Info struct {
	ID       uint64
	Time     string
	Module   string
	Level    LogLevel
	Line     int
	Filename string
	Message  string
	//format   string
}

// Logger class that is an interface to user to log messages, Module is the module for which we are testing
// worker is variable of Worker class that is used in bottom layers to log the message
type Logger struct {
	Module string
	worker *Worker
}

// init pkg
func init() {
	initColors()
	initFormatPlaceholders()
}

// Output returns a proper string to be outputted for a particular info
func (r *Info) Output(format string) string {
	msg := fmt.Sprintf(format,
		r.ID,               // %[1] // %{id}
		r.Time,             // %[2] // %{time[:fmt]}
		r.Module,           // %[3] // %{module}
		r.Filename,         // %[4] // %{filename}
		r.Line,             // %[5] // %{line}
		r.logLevelString(), // %[6] // %{level}
		r.Message,          // %[7] // %{message}
	)
	// Ignore printf errors if len(args) > len(verbs)
	if i := strings.LastIndex(msg, "%!(EXTRA"); i != -1 {
		return msg[:i]
	}
	return msg
}

// Analyze and represent format string as printf format string and time format
func parseFormat(format string) (msgfmt, timefmt string) {
	if len(format) < 10 /* (len of "%{message} */ {
		return defFmt, defTimeFmt
	}
	timefmt = defTimeFmt
	idx := strings.IndexRune(format, '%')
	for idx != -1 {
		msgfmt += format[:idx]
		format = format[idx:]
		if len(format) > 2 {
			if format[1] == '{' {
				// end of curr verb pos
				if jdx := strings.IndexRune(format, '}'); jdx != -1 {
					// next verb pos
					idx = strings.Index(format[1:], "%{")
					// incorrect verb found ("...%{wefwef ...") but after
					// this, new verb (maybe) exists ("...%{inv %{verb}...")
					if idx != -1 && idx < jdx {
						msgfmt += "%%"
						format = format[1:]
						continue
					}
					// get verb and arg
					verb, arg := ph2verb(format[:jdx+1])
					msgfmt += verb
					// check if verb is time
					// here you can handle args for other verbs
					if verb == `%[2]s` && arg != "" /* %{time} */ {
						timefmt = arg
					}
					format = format[jdx+1:]
				} else {
					format = format[1:]
				}
			} else {
				msgfmt += "%%"
				format = format[1:]
			}
		}
		idx = strings.IndexRune(format, '%')
	}
	msgfmt += format
	return
}

// translate format placeholder to printf verb and some argument of placeholder
// (now used only as time format)
func ph2verb(ph string) (verb string, arg string) {
	n := len(ph)
	if n < 4 {
		return ``, ``
	}
	if ph[0] != '%' || ph[1] != '{' || ph[n-1] != '}' {
		return ``, ``
	}
	idx := strings.IndexRune(ph, ':')
	if idx == -1 {
		return phfs[ph], ``
	}
	verb = phfs[ph[:idx]+"}"]
	arg = ph[idx+1 : n-1]
	return
}

// NewWorker returns an instance of worker class, prefix is the string attached to every log,
// flag determine the log params, color parameters verifies whether we need colored outputs or not
func NewWorker(prefix string, flag int, color int, out io.Writer) *Worker {
	return &Worker{Minion: log.New(out, prefix, flag), Color: color, format: defFmt, timeFormat: defTimeFmt}
}

// SetDefaultFormat sets default format for the message
func SetDefaultFormat(format string) {
	defFmt, defTimeFmt = parseFormat(format)
}

// SetFormat for the worker sets the format for the worker
func (w *Worker) SetFormat(format string) {
	w.format, w.timeFormat = parseFormat(format)
}

// SetFormat for teh logger sets format for the logger
func (l *Logger) SetFormat(format string) {
	l.worker.SetFormat(format)
}

// SetLogLevel for the worker sets the log level for the worker
func (w *Worker) SetLogLevel(level LogLevel) {
	w.level = level
}

// SetLogLevel for the logger sets the log level for the logger
func (l *Logger) SetLogLevel(level LogLevel) {
	l.worker.level = level
}

// Log is a function of Worker class to log a string based on level
func (w *Worker) Log(level LogLevel, calldepth int, info *Info) error {

	if w.level < level {
		return nil
	}

	if w.Color != 0 {
		buf := &bytes.Buffer{}
		buf.Write([]byte(colors[level]))
		buf.Write([]byte(info.Output(w.format)))
		buf.Write([]byte("\033[0m"))
		return w.Minion.Output(calldepth+1, buf.String())
	}
	return w.Minion.Output(calldepth+1, info.Output(w.format))

}

// Returns a proper string to output for colored logging
func colorString(color int) string {
	return fmt.Sprintf("\033[%dm", int(color))
}

// Initializes the map of colors
func initColors() {
	colors = map[LogLevel]string{
		CriticalLevel: colorString(Magenta),
		ErrorLevel:    colorString(Red),
		WarningLevel:  colorString(Yellow),
		NoticeLevel:   colorString(Green),
		DebugLevel:    colorString(Cyan),
		InfoLevel:     colorString(White),
	}
}

// Initializes the map of placeholders
func initFormatPlaceholders() {
	phfs = map[string]string{
		"%{id}":       "%[1]d",
		"%{time}":     "%[2]s",
		"%{module}":   "%[3]s",
		"%{filename}": "%[4]s",
		"%{file}":     "%[4]s",
		"%{line}":     "%[5]d",
		"%{level}":    "%[6]s",
		"%{lvl}":      "%.3[6]s",
		"%{message}":  "%[7]s",
	}
}

// New returns a new instance of logger class, module is the specific module for which we are logging
// , color defines whether the output is to be colored or not, out is instance of type io.Writer defaults
// to os.Stderr
func New(args ...interface{}) (*Logger, error) {
	//initColors()

	var module string = "DEFAULT"
	var color int = 1
	var out io.Writer = os.Stderr
	var level LogLevel = InfoLevel

	for _, arg := range args {
		switch t := arg.(type) {
		case string:
			module = t
		case int:
			color = t
		case io.Writer:
			out = t
		case LogLevel:
			level = t
		default:
			panic("logger: Unknown argument")
		}
	}
	newWorker := NewWorker("", 0, color, out)
	newWorker.SetLogLevel(level)
	return &Logger{Module: module, worker: newWorker}, nil
}

// Log commnand is the function available to user to log message, lvl specifies
// the degree of the messagethe user wants to log, message is the info user wants to log
func (l *Logger) Log(lvl LogLevel, message string) {
	l.logInternal(lvl, message, 2)
}

func (l *Logger) logInternal(lvl LogLevel, message string, pos int) {
	//var formatString string = "#%d %s [%s] %s:%d ▶ %.3s %s"
	_, filename, line, _ := runtime.Caller(pos)
	filename = path.Base(filename)
	info := &Info{
		ID:       atomic.AddUint64(&logNo, 1),
		Time:     time.Now().Format(l.worker.timeFormat),
		Module:   l.Module,
		Level:    lvl,
		Message:  message,
		Filename: filename,
		Line:     line,
		//format:   formatString,
	}
	l.worker.Log(lvl, 2, info)
}

// Fatal is just like func l.Critical logger except that it is followed by exit to program
func (l *Logger) Fatal(message string) {
	l.logInternal(CriticalLevel, message, 2)
	os.Exit(1)
}

// Fatalf is just like func l.CriticalF logger except that it is followed by exit to program
func (l *Logger) Fatalf(format string, a ...interface{}) {
	l.logInternal(CriticalLevel, fmt.Sprintf(format, a...), 2)
	os.Exit(1)
}

// Panic is just like func l.Critical except that it is followed by a call to panic
func (l *Logger) Panic(message string) {
	l.logInternal(CriticalLevel, message, 2)
	panic(message)
}

// Panicf is just like func l.CriticalF except that it is followed by a call to panic
func (l *Logger) Panicf(format string, a ...interface{}) {
	l.logInternal(CriticalLevel, fmt.Sprintf(format, a...), 2)
	panic(fmt.Sprintf(format, a...))
}

// Critical logs a message at a Critical Level
func (l *Logger) Critical(message string) {
	l.logInternal(CriticalLevel, message, 2)
}

// Criticalf logs a message at Critical level using the same syntax and options as fmt.Printf
func (l *Logger) Criticalf(format string, a ...interface{}) {
	l.logInternal(CriticalLevel, fmt.Sprintf(format, a...), 2)
}

// Error logs a message at Error level
func (l *Logger) Error(message string) {
	l.logInternal(ErrorLevel, message, 2)
}

// Errorf logs a message at Error level using the same syntax and options as fmt.Printf
func (l *Logger) Errorf(format string, a ...interface{}) {
	l.logInternal(ErrorLevel, fmt.Sprintf(format, a...), 2)
}

// Warning logs a message at Warning level
func (l *Logger) Warning(message string) {
	l.logInternal(WarningLevel, message, 2)
}

// Warningf logs a message at Warning level using the same syntax and options as fmt.Printf
func (l *Logger) Warningf(format string, a ...interface{}) {
	l.logInternal(WarningLevel, fmt.Sprintf(format, a...), 2)
}

// Notice logs a message at Notice level
func (l *Logger) Notice(message string) {
	l.logInternal(NoticeLevel, message, 2)
}

// Noticef logs a message at Notice level using the same syntax and options as fmt.Printf
func (l *Logger) Noticef(format string, a ...interface{}) {
	l.logInternal(NoticeLevel, fmt.Sprintf(format, a...), 2)
}

// Info logs a message at Info level
func (l *Logger) Info(message string) {
	l.logInternal(InfoLevel, message, 2)
}

// Infof logs a message at Info level using the same syntax and options as fmt.Printf
func (l *Logger) Infof(format string, a ...interface{}) {
	l.logInternal(InfoLevel, fmt.Sprintf(format, a...), 2)
}

// Debug logs a message at Debug level
func (l *Logger) Debug(message string) {
	l.logInternal(DebugLevel, message, 2)
}

//Debugf logs a message at Debug level using the same syntax and options as fmt.Printf
func (l *Logger) Debugf(format string, a ...interface{}) {
	l.logInternal(DebugLevel, fmt.Sprintf(format, a...), 2)
}

// StackAsError prints this goroutine's execution stack as an error with an optional message at the begining
func (l *Logger) StackAsError(message string) {
	if message == "" {
		message = "Stack info"
	}
	message += "\n"
	l.logInternal(ErrorLevel, message+Stack(), 2)
}

// StackAsCritical prints this goroutine's execution stack as critical with an optional message at the begining
func (l *Logger) StackAsCritical(message string) {
	if message == "" {
		message = "Stack info"
	}
	message += "\n"
	l.logInternal(CriticalLevel, message+Stack(), 2)
}

// Stack returns a string with the execution stack for this goroutine
func Stack() string {
	buf := make([]byte, 1000000)
	runtime.Stack(buf, false)
	return string(buf)
}

// Returns the loglevel as string
func (r *Info) logLevelString() string {
	logLevels := [...]string{
		"CRITICAL",
		"ERROR",
		"WARNING",
		"NOTICE",
		"INFO",
		"DEBUG",
	}
	return logLevels[r.Level-1]
}
