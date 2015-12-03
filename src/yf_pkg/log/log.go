package log

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

const (
	ERROR      = 0
	ERROR_STR  = "error"
	WARN       = 1
	WARN_STR   = "warn"
	NOTICE     = 2
	NOTICE_STR = "notice"
	DEBUG      = 3
	DEBUG_STR  = "debug"
)

type Log struct {
	level   int
	content string
}

var levelStr = []string{"ERROR", "WARN", "NOTICE", "DEBUG"}
var levelSStr = []string{ERROR_STR, WARN_STR, NOTICE_STR, DEBUG_STR}
var levelStrMap map[string]int = map[string]int{ERROR_STR: ERROR, DEBUG_STR: DEBUG, NOTICE_STR: NOTICE, WARN_STR: WARN}

type Logger struct {
	logChan  chan Log
	cron     bool
	handler  *log.Logger
	fileName string
	file     *os.File
	level    int
}

func NewLog(level int, content string) (log Log) {
	log.level = level
	log.content = content
	return
}

func New2(file string, bufferSize int, level string) (l *Logger, err error) {
	levelInt, ok := levelStrMap[level]
	if !ok {
		return nil, errors.New("unknown log level : " + level)
	}
	return New(file, bufferSize, levelInt)
}

func New(file string, bufferSize int, level int) (l *Logger, err error) {
	if level > DEBUG {
		err = errors.New("invalid log level")
		return nil, err
	}
	l = new(Logger)
	l.level = level
	l.fileName = file
	l.cron = false

	l.file, err = os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	l.handler = log.New(l.file, "", log.LstdFlags)
	//l.handler = log.New(l.file, "", log.LstdFlags|log.Lshortfile)
	l.logChan = make(chan Log, bufferSize)
	go l.write()
	go l.checkCronlog()
	return l, nil
}

func (l *Logger) Close() error {
	close(l.logChan)
	return nil
}

func (l *Logger) Append(log string, level ...int) {
	item := NewLog(ERROR, log)
	if len(level) > 0 {
		//	fmt.Printf("request level : %v, accept level :%v\n", level[0], l.level)
		item.level = level[0]
	}
	if item.level <= l.level {
		l.logChan <- item
	}
}

func (l *Logger) AppendInfo(args ...interface{}) {
	s := ""
	for _, v := range args {
		s += fmt.Sprintf(" | %v | ", v)
	}
	l.Append(s, DEBUG)
}

func (l *Logger) AppendObj(err error, args ...interface{}) {
	edesc := ""
	level := DEBUG
	if err != nil {
		edesc = err.Error()
		level = ERROR
	}
	s := ""
	for _, v := range args {
		s += fmt.Sprintf(" | %v | ", v)
	}
	r := s + edesc
	l.Append(r, level)
}

//检查是否需要滚动日志
func (l *Logger) checkCronlog() {
	for {
		if _, err := os.Open(l.fileName); l.cron == false && os.IsNotExist(err) {
			l.cron = true
		}
		time.Sleep(10 * time.Second)
	}
}

func (l *Logger) write() {
	for v := range l.logChan {
		if l.cron {
			l.file.Close()
			var err error
			l.file, err = os.OpenFile(l.fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				panic(err.Error())
			}
			l.handler = log.New(l.file, "", log.LstdFlags)
			l.cron = false
		}
		l.handler.Println("[", levelStr[v.level], "] ", v.content)
	}
	l.file.Close()
}

func LevelToString(level int) string {
	if level <= DEBUG {
		return levelStr[level]
	} else {
		return "UNKNOWN"
	}
}
