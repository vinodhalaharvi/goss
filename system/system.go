package system

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"

	"github.com/aelsabbahy/GOnetstat"
	// This needs a better name
	"github.com/aelsabbahy/go-ps"
	"github.com/urfave/cli"
	util2 "github.com/vinodhalaharvi/goss/util"
)

type Resource interface {
	Exists() (bool, error)
}

type System struct {
	NewPort      func(string, *System, util2.Config) Port
	NewService   func(string, *System, util2.Config) Service
	NewCommand   func(string, *System, util2.Config) Command
	NewProcess   func(string, *System, util2.Config) Process
	NewGossfile  func(string, *System, util2.Config) Gossfile
	NewInterface func(string, *System, util2.Config) Interface
	ports        map[string][]GOnetstat.Process
	portsOnce    sync.Once
	procMap      map[string][]ps.Process
	procOnce     sync.Once
}

func (s *System) Ports() map[string][]GOnetstat.Process {
	s.portsOnce.Do(func() {
		s.ports = GetPorts(false)
	})
	return s.ports
}

func (s *System) ProcMap() map[string][]ps.Process {
	s.procOnce.Do(func() {
		s.procMap = GetProcs()
	})
	return s.procMap
}

func New(c *cli.Context) *System {
	sys := &System{
		NewPort:      NewDefPort,
		NewCommand:   NewDefCommand,
		NewProcess:   NewDefProcess,
		NewGossfile:  NewDefGossfile,
		NewInterface: NewDefInterface,
	}
	sys.detectService()
	return sys
}

// detectService adds the correct service creation function to a System struct
func (sys *System) detectService() {
	switch DetectService() {
	case "systemd":
		sys.NewService = NewServiceSystemd
	}
}

// DetectPackageManager attempts to detect whether or not the system is using
// "deb", "rpm", "apk", or "pacman" package managers. It first attempts to
// detect the distro. If that fails, it falls back to finding package manager
// executables. If that fails, it returns the empty string.

// DetectService attempts to detect what kind of service management the system
// is using, "systemd", "upstart", "alpineinit", or "init". It looks for systemctl
// command to detect systemd, and falls back on DetectDistro otherwise. If it can't
// decide, it returns "init".
func DetectService() string {
	if HasCommand("systemctl") {
		return "systemd"
	}
	// Centos Docker container doesn't run systemd, so we detect it or use init.
	switch DetectDistro() {
	case "ubuntu":
		return "upstart"
	case "alpine":
		return "alpineinit"
	case "arch":
		return "systemd"
	}
	return "init"
}

// DetectDistro attempts to detect which Linux distribution this computer is
// using. One of "ubuntu", "redhat" (including Centos), "alpine", "arch", or
// "debian". If it can't decide, it returns an empty string.
func DetectDistro() string {
	if b, e := ioutil.ReadFile("/etc/lsb-release"); e == nil && bytes.Contains(b, []byte("Ubuntu")) {
		return "ubuntu"
	} else if isRedhat() {
		return "redhat"
	} else if _, err := os.Stat("/etc/alpine-release"); err == nil {
		return "alpine"
	} else if _, err := os.Stat("/etc/arch-release"); err == nil {
		return "arch"
	} else if _, err := os.Stat("/etc/debian_version"); err == nil {
		return "debian"
	}
	return ""
}

// HasCommand returns whether or not an executable by this name is on the PATH.
func HasCommand(cmd string) bool {
	if _, err := exec.LookPath(cmd); err == nil {
		return true
	}
	return false
}

func isRedhat() bool {
	if _, err := os.Stat("/etc/redhat-release"); err == nil {
		return true
	} else if _, err := os.Stat("/etc/system-release"); err == nil {
		return true
	}
	return false
}
