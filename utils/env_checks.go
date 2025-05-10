package utils

import (
        "fmt"
        tea "github.com/charmbracelet/bubbletea"
        "os"
)

func check(e error) {
        if e != nil {
                panic(e)
        }
}

func EnvChecks {

}

func makeTempDir() {
        tmpdir, err := os.MkdirTemp("", "tmp.*")
        check(err)
        fmt.Println("Temp dir created -", tmpdir)
}

func EnvChecks() {
        // Check for temp directory
        if os. 
}
