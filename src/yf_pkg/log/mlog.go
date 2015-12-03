package log

import "errors"

type MLogger []*Logger

func NewMLogger(prefix string, bufferSize int, level string) (l *MLogger, err error) {
	levelInt, ok := levelStrMap[level]
	if !ok {
		return nil, errors.New("unknown log level : " + level)
	}
	if levelInt > DEBUG {
		err = errors.New("invalid log level")
		return nil, err
	}
	mlog := make(MLogger, levelInt+1)
	l = &mlog
	for i, _ := range mlog {
		mlog[i], err = New(prefix+"."+levelSStr[i]+".log", bufferSize, i)
		if err != nil {
			return nil, err
		}
	}
	return
}
func (l *MLogger) Append(log string, levels ...int) {
	level := ERROR
	if len(levels) > 0 {
		level = levels[0]
	}
	if level < len(*l) {
		(*l)[level].Append(log, level)
	}
}
func (l *MLogger) AppendObj(err error, args ...interface{}) {
	if err != nil {
		(*l)[ERROR].AppendObj(err, args...)
	} else if len(*l) > DEBUG {
		(*l)[DEBUG].AppendObj(err, args...)
	}
}

func (l *MLogger) AppendInfo(args ...interface{}) {
	if len(*l) > DEBUG {
		(*l)[DEBUG].AppendInfo(args...)
	}
}
