package runconfig

import flag "github.com/docker/docker/pkg/mflag"

func ParseRunIn(cmd *flag.FlagSet, args []string) (*RunInConfig, error) {
	var (
		flPrivileged = cmd.Bool([]string{"#privileged", "-privileged"}, false, "Give extended privileges to this container")
		flStdin      = cmd.Bool([]string{"i", "-interactive"}, false, "Keep STDIN open even if not attached")
		flTty        = cmd.Bool([]string{"t", "-tty"}, false, "Allocate a pseudo-TTY")
		flHostname   = cmd.String([]string{"h", "-hostname"}, "", "Container host name")
		flUser       = cmd.String([]string{"u", "-user"}, "", "Username or UID")
		flDetach     = cmd.Bool([]string{"d", "-detach"}, false, "Detached mode: run command in the background")
		runInCmd     []string
		container    string
	)
	if err := cmd.Parse(args); err != nil {
		return nil, err
	}
	parsedArgs := cmd.Args()
	if len(parsedArgs) > 1 {
		container = cmd.Arg(0)
		runInCmd = parsedArgs[1:]
	}

	runInConfig := &RunInConfig{
		User:       *flUser,
		Privileged: *flPrivileged,
		Tty:        *flTty,
		Cmd:        runInCmd,
		Container:  container,
		Hostname:   *flHostname,
		Detach: *flDetach,
	}

	// If -d is not set, attach to everything by default
	if !*flDetach {
		runInConfig.AttachStdout = true
		runInConfig.AttachStderr = true
		if *flStdin {
			runInConfig.AttachStdin = true
		}
	}

	return runInConfig, nil
}
