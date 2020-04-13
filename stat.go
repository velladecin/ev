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
    frequencyDuration       time.Duration
    callback                func()
}

func (fso *FSObject) Notify() {
}

func (fso *FSObject) Watch() {
    go func() {
        for {
            start := time.Now()

            exists, isFile, ctime, err := stat(fso.path)
            if (err != nil) {
                // do nothing
                // TODO log
                fmt.Printf("stat() on %s failed with: %s", fso.path, err)
            } else {
                callCallback := false

                if exists != fso.exists {
                    fso.exists = exists
                    callCallback = true
                }

                if isFile != fso.isFile {
                    fso.isFile= isFile
                    callCallback = true
                }

                if ctime != fso.ctime {
                    fso.ctime = ctime
                    callCallback = true
                }

                if callCallback {
                    fmt.Println(">>>>>>>>>>>>> Callback>>>>>>")
                    fso.callback()
                }
            }

            end := time.Now()
            elapsed := end.Sub(start)
            time.Sleep(fso.frequencyDuration - elapsed)
        }
    }()
}

func Stat(path string, frequency int, strict bool, fn func()) (*FSObject, error) {
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
