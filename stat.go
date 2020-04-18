package ev

import (
    "fmt"
    "os"
    "syscall"
    "time"
)

const (
    FREQUENCY = 2 // seconds
)

type FSObject struct {
    path                    string
    exists, isFile, strict  bool            // file, dir only
    ctime                   int64
    frequency               int
    frequencyDuration       time.Duration   // frequency converted to Duration
    callback                func(string)
}

/*
type FSNotify struct {
    path                    string
    exists, isFile          bool
    ctime                   int64
}
*/

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
                fso.callback(fso.path)
            }
        }
    }()
}

func (fso *FSObject) watch(ch chan bool) {
    for {
        start := time.Now()

        exists, isFile, ctime, err := stat(fso.path)
        if (err != nil) {
            // do nothing
            // TODO log
            fmt.Printf("stat() on %s failed with: %s", fso.path, err)
        } else {
            callback := false

            if exists != fso.exists {
                fso.exists = exists
                callback = true
            }

            if isFile != fso.isFile {
                fso.isFile= isFile
                callback = true
            }

            if ctime != fso.ctime {
                fso.ctime = ctime
                callback = true
            }

            if callback {
                ch <- true
            }
        }

        end := time.Now()
        elapsed := end.Sub(start)
        time.Sleep(fso.frequencyDuration - elapsed)
    }
}

func Stat(path string, frequency int, strict bool, fn func(string)) (*FSObject, error) {
    if (frequency < 1) {
        frequency = FREQUENCY
    }

    exists, isFile, ctime, err := stat(path)
    if err != nil {
        return nil, err
    }

    return &FSObject {
        path: path,
        exists: exists,
        isFile: isFile,
        strict: strict,
        ctime: ctime,
        frequency: frequency,
        frequencyDuration: time.Duration(frequency) * time.Second,
        callback: fn,
    }, nil
}

func stat(path string) (bool, bool, int64, error) {
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
            return isFile, exists, ctime, fmt.Errorf("Accepting file/dir, got(%s)", t)
        }

        s := fi.Sys().(*syscall.Stat_t)
        ctime = int64(s.Ctim.Sec)
    }

    return exists, isFile, ctime, nil
}
