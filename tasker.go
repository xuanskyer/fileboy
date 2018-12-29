package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Tasker struct {
	lastTaskId int64
	delay      int
	cmd        *exec.Cmd
	putLock    sync.Mutex
	runLock    sync.Mutex
}

func newTasker(delay int) *Tasker {
	return &Tasker{
		delay: delay,
	}
}

func (t *Tasker) Put(cf *changeFile) {
	if t.delay < 1 {
		go t.run(cf)
		return
	}
	t.putLock.Lock()
	defer t.putLock.Unlock()
	t.lastTaskId = cf.Changed
	go func() {
		tc := time.Tick(time.Millisecond * time.Duration(t.delay))
		<-tc
		if t.lastTaskId > cf.Changed {
			return
		}
		t.preRun(cf)
	}()
}

func (t *Tasker) preRun(cf *changeFile) {
	if t.cmd != nil {
		err := t.cmd.Process.Kill()
		if err != nil {
			log.Println("err: ", err)
		}
		log.Println("stop old process ")
	}
	go t.run(cf)
}

func (t *Tasker) run(cf *changeFile) {
	t.runLock.Lock()
	defer t.runLock.Unlock()
	for i := 0; i < len(cfg.Command.Exec); i++ {
		carr := cmdParse2Array(cfg.Command.Exec[i], cf)
		log.Println(carr)
		t.cmd = exec.Command(carr[0], carr[1:]...)
		//cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_UNICODE_ENVIRONMENT}
		t.cmd.Stdin = os.Stdin
		//cmd.Stdout = os.Stdout
		t.cmd.Stderr = os.Stderr
		t.cmd.Dir = projectFolder
		t.cmd.Env = os.Environ()
		stdout, err := t.cmd.StdoutPipe()
		if err != nil {
			log.Println("error=>", err.Error())
			return
		}
		_ = t.cmd.Start()
		reader := bufio.NewReader(stdout)
		for {
			line, err2 := reader.ReadString('\n')
			if err2 != nil || io.EOF == err2 {
				break
			}
			log.Print(line)
		}
		err = t.cmd.Wait()
		if err != nil {
			log.Println("cmd wait err ", err)
			break
		}
		err = t.cmd.Process.Kill()
		if err != nil {
			log.Println("cmd cannot kill ", err)
		}
	}

	log.Println("end ")
}
