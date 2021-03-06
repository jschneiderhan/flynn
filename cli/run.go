package main

import (
	"errors"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/flynn/flynn/controller/client"
	ct "github.com/flynn/flynn/controller/types"
	"github.com/flynn/flynn/pkg/demultiplex"
	"github.com/flynn/flynn/Godeps/_workspace/src/github.com/heroku/hk/term"
)

var (
	runDetached bool
	runRelease  string
)

var cmdRun = &Command{
	Run:   runRun,
	Usage: "run [-d] [-r <release>] <command> [<argument>...]",
	Short: "run a job",
	Long:  `Run a job`,
}

func init() {
	cmdRun.Flag.BoolVarP(&runDetached, "detached", "d", false, "run job without connecting io streams")
	cmdRun.Flag.StringVarP(&runRelease, "release", "r", "", "id of release to run (defaults to current app release)")
}

func runRun(cmd *Command, args []string, client *controller.Client) error {
	if len(args) == 0 {
		cmd.printUsage(true)
	}
	if runRelease == "" {
		release, err := client.GetAppRelease(mustApp())
		if err == controller.ErrNotFound {
			return errors.New("No app release, specify a release with -release")
		}
		if err != nil {
			return err
		}
		runRelease = release.ID
	}
	req := &ct.NewJob{
		Cmd:       args,
		TTY:       term.IsTerminal(os.Stdin) && term.IsTerminal(os.Stdout) && !runDetached,
		ReleaseID: runRelease,
	}
	if req.TTY {
		cols, err := term.Cols()
		if err != nil {
			return err
		}
		lines, err := term.Lines()
		if err != nil {
			return err
		}
		req.Columns = cols
		req.Lines = lines
		req.Env = map[string]string{
			"COLUMNS": strconv.Itoa(cols),
			"LINES":   strconv.Itoa(lines),
			"TERM":    os.Getenv("TERM"),
		}
	}

	if runDetached {
		job, err := client.RunJobDetached(mustApp(), req)
		if err != nil {
			return err
		}
		log.Println(job.ID)
		return nil
	}

	rwc, err := client.RunJobAttached(mustApp(), req)
	if err != nil {
		return err
	}
	defer rwc.Close()

	if req.TTY {
		if err := term.MakeRaw(os.Stdin); err != nil {
			return err
		}
		defer term.Restore(os.Stdin)
	}

	go func() {
		io.Copy(rwc, os.Stdin)
		rwc.CloseWrite()
	}()
	if req.TTY {
		_, err = io.Copy(os.Stdout, rwc)
	} else {
		err = demultiplex.Copy(os.Stdout, os.Stderr, rwc)
	}
	// TODO: get exit code and use it
	return err
}
