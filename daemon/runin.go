package daemon

import (
	"io"

	"github.com/docker/docker/daemon/execdriver"
	"github.com/docker/docker/engine"
	"github.com/docker/docker/runconfig"
)

func (daemon *Daemon) ContainerRunIn(job *engine.Job) engine.Status {
	if len(job.Args) < 2 {
		return job.Errorf("Usage: %s container_id command", job.Name)
	}
	var (
		cStdin           io.ReadCloser
		cStdout, cStderr io.Writer
		cStdinCloser     io.Closer
		name             = job.Args[0]
	)

	runInConfig := runconfig.RunInConfigFromJob(job)

	if runInConfig.Stdin {
		r, w := io.Pipe()
		go func() {
			defer w.Close()
			io.Copy(w, job.Stdin)
		}()
		cStdin = r
		cStdinCloser = job.Stdin
	}
	if runInConfig.Stdout {
		cStdout = job.Stdout
	}
	if runInConfig.Stderr {
		cStderr = job.Stderr
	}

	if err := daemon.RunInContainer(runInConfig, name, func(stdConfig *daemon.StdConfig) chan error {
		return daemon.NewAttach(stdConfig, runInConfig.AttachStdin, false, runInConfig.Tty, cStdin, cStdinCloser, cStdout, cStderr)
	}); err != nil {
		return job.Error(err)
	}
	daemon.LogEvent("runin", name, "")
	return engine.StatusOK
}

func (daemon *Daemon) RunIn(c *Container, runInConfig *RunInConfig, pipes *execdriver.Pipes, startCallback execdriver.StartCallback) (int, error) {
	return daemon.execDriver.RunIn(c.command, runInConfig.ProcessConfig, pipes, startCallback)
}

func (daemon *Daemon) RunInContainer(config *runconfig.RunInConfig, name string, attachCallback func(*StdConfig) chan error) error {
	container := daemon.Get(name)
	if container == nil {
		return fmt.Errorf("No such container: %s", name)
	}

	if container.State.IsRunning() {
		return fmt.Errorf("Container already started")
	}

	entrypoint, args := daemon.getEntrypointAndArgs(nil, config.Cmd)

	processConfig := &execdriver.ProcessConfig{
		Privileged: config.Privileged,
		User:       config.User,
		Tty:        config.Tty,
		Entrypoint: entrypoint,
		Arguments:  args,
	}

	runInConfig := &RunInConfig{
		OpenStdin:     config.AttachStdin,
		StdConfig:     &StdConfig{},
		ProcessConfig: processConfig,
	}

	runInConfig.StdConfig.stderr = broadcastwriter.New()
	runInConfig.StdConfig.stdout = broadcastwriter.New()
	// Attach to stdin
	if runInConfig.OpenStdin {
		runInConfig.StdConfig.stdin, runInConfig.StdConfig.stdinPipe = io.Pipe()
	} else {
		runInConfig.StdConfig.stdinPipe = utils.NopWriteCloser(ioutil.Discard) // Silently drop stdin
	}
	var errChan chan error
	go func() {
		errChan = attachCallback(&runInConfig.StdConfig)
		utils.Debugf("Run In callback done")
	}()
	utils.Debugf("About to run container RunIn with config %+v\n", runInConfig)
	VishLog.Printf("About to run container RunIn with config %+v\n", runInConfig)
	if err := container.RunIn(runInConfig); err != nil {
		utils.Debugf("container Run In failed - %s\n", err)
		return fmt.Errorf("Cannot run in container %s: %s", name, err)
	}	
	utils.Debugf("daemon.go RunIn completed.")

	return <-errChan
}
