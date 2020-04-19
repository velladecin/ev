package ev

import (
    "os"
    "syscall"
    "time"
    "log"
)

const (
    FREQUENCY = 2 // seconds
)

type FSObject struct {
    path                    string
    exists, isFile, strict  bool            // file, dir only
    ctime                   int64
    frequency               time.Duration
    callback                func([]interface{})
}

func NewStatNotify(path string, frequency int, strict bool) *FSObject {
    return NewStat(path, frequency, strict, func([]interface{}){})
}

func NewStat(path string, frequency int, strict bool, fn func([]interface{})) *FSObject {
    if (frequency < 1) {
        frequency = FREQUENCY
    }

    exists, isFile, ctime := stat(path)

    return &FSObject {
        path: path,
        exists: exists,
        isFile: isFile,
        strict: strict,
        ctime: ctime,
        frequency: time.Duration(frequency) * time.Second,
        callback: fn,
    }
}

func (fso *FSObject) Notify(notify chan bool) {
    ch := make(chan bool)

    go fso.watch(ch)
    go func() {
        for changed := range ch {
            if changed {
                notify <- changed
            }
        }
    }()
}

func (fso *FSObject) Run() {
    ch := make(chan bool)

    go fso.watch(ch)
    go func() {
        for changed := range ch {
            if changed {
                fso.callback([]interface{}{fso.path})
            }
        }
    }()
}

func (fso *FSObject) watch(ch chan bool) {
    for {
        start := time.Now()

        exists, isFile, ctime := stat(fso.path)

        /* ctime change would cover changes to all the other attributes
           so check just ctime.. */

        if ctime != fso.ctime {
            fso.ctime = ctime
            fso.exists = exists
            fso.isFile = isFile
            // and nofify..
            ch <- true
        }

        end := time.Now()
        elapsed := end.Sub(start)
        time.Sleep(fso.frequency - elapsed)
    }
}

func stat(path string) (bool, bool, int64) {
    // not found (default)
    var exists, isFile bool
    ctime := int64(0)

    // found
    if fi, err := os.Stat(path); err == nil {
        exists = true

        var t string
        switch m := fi.Mode(); {
            // good
            case m.IsRegular(): isFile = true
            case m.IsDir():     isFile = false
            // bad
            case m & os.ModeSymlink     != 0: t = "symlink"
            case m & os.ModeNamedPipe   != 0: t = "named pipe (FIFO)"
            case m & os.ModeSocket      != 0: t = "socket"
            case m & os.ModeCharDevice  != 0: t = "character device"
            case m & os.ModeIrregular   != 0: t = "unknown file"
            default:
                t = "unknown file type"
        }

        if t != "" {
            log.Fatalf("Accepting file/dir, got(%s)", t)
        }

        s := fi.Sys().(*syscall.Stat_t)
        ctime = int64(s.Ctim.Sec)
    }

    return exists, isFile, ctime
}
