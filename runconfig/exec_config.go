package runconfig

import "github.com/docker/docker/engine"

type ExecConfig struct {
	User         string
	Privileged   bool
	Tty          bool
	Container    string
	AttachStdin  bool
	AttachStderr bool
	AttachStdout bool
	Detach       bool
	Cmd          []string
	Hostname     string
}

func ExecConfigFromJob(job *engine.Job) *ExecConfig {
	execConfig := &ExecConfig{
		User:         job.Getenv("User"),
		Privileged:   job.GetenvBool("Privileged"),
		Tty:          job.GetenvBool("Tty"),
		Container:    job.Getenv("Container"),
		AttachStdin:  job.GetenvBool("AttachStdin"),
		AttachStderr: job.GetenvBool("AttachStderr"),
		AttachStdout: job.GetenvBool("AttachStdout"),
	}
	if Cmd := job.GetenvList("Cmd"); Cmd != nil {
		execConfig.Cmd = Cmd
	}

	return execConfig
}
