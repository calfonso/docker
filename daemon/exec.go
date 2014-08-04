package daemon

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/docker/docker/daemon/execdriver"
	"github.com/docker/docker/engine"
	"github.com/docker/docker/runconfig"
	"github.com/docker/docker/utils"
	"github.com/docker/docker/utils/broadcastwriter"
)

func (d *Daemon) ContainerExec(job *engine.Job) engine.Status {
	if len(job.Args) != 1 {
		return job.Errorf("Usage: %s container_id command", job.Name)
	}

	var (
		cStdin           io.ReadCloser
		cStdout, cStderr io.Writer
		cStdinCloser     io.Closer
		name             = job.Args[0]
	)

	container := d.Get(name)

	if container == nil {
		return job.Errorf("No such container: %s", name)
	}

	if !container.State.IsRunning() {
		return job.Errorf("Container %s is not not running", name)
	}

	config := runconfig.ExecConfigFromJob(job)

	if config.AttachStdin {
		r, w := io.Pipe()
		go func() {
			defer w.Close()
			io.Copy(w, job.Stdin)
		}()
		cStdin = r
		cStdinCloser = job.Stdin
	}
	if config.AttachStdout {
		cStdout = job.Stdout
	}
	if config.AttachStderr {
		cStderr = job.Stderr
	}

	entrypoint, args := d.getEntrypointAndArgs(nil, config.Cmd)

	processConfig := execdriver.ProcessConfig{
		Privileged: config.Privileged,
		User:       config.User,
		Tty:        config.Tty,
		Entrypoint: entrypoint,
		Arguments:  args,
	}

	execConfig := &ExecConfig{
		OpenStdin:     config.AttachStdin,
		StdConfig:     StdConfig{},
		ProcessConfig: processConfig,
	}

	execConfig.StdConfig.stderr = broadcastwriter.New()
	execConfig.StdConfig.stdout = broadcastwriter.New()
	// Attach to stdin
	if execConfig.OpenStdin {
		execConfig.StdConfig.stdin, execConfig.StdConfig.stdinPipe = io.Pipe()
	} else {
		execConfig.StdConfig.stdinPipe = utils.NopWriteCloser(ioutil.Discard) // Silently drop stdin
	}

	var execErr, attachErr chan error
	go func() {
		attachErr = d.Attach(&execConfig.StdConfig, config.AttachStdin, false, config.Tty, cStdin, cStdinCloser, cStdout, cStderr)
	}()

	go func() {
		err := container.Exec(execConfig)
		if err != nil {
			err = fmt.Errorf("Cannot run in container %s: %s", name, err)
		}
		execErr <- err
	}()

	select {
	case err := <-attachErr:
		return job.Errorf("attach failed with error: %s", err)
	case err := <-execErr:
		return job.Error(err)
	}

	return engine.StatusOK
}

func (daemon *Daemon) Exec(c *Container, execConfig *ExecConfig, pipes *execdriver.Pipes, startCallback execdriver.StartCallback) (int, error) {
	return daemon.execDriver.Exec(c.command, &execConfig.ProcessConfig, pipes, startCallback)
}
