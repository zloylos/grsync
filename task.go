package grsync

import (
	"bufio"
	"io"
	"math"
	"strconv"
	"strings"
)

// Task is high-level API under rsync
type Task struct {
	rsync *Rsync

	state *State
	log   *Log
}

// State contains information about rsync process
type State struct {
	Remain   int     `json:"remain"`
	Total    int     `json:"total"`
	Speed    string  `json:"speed"`
	Progress float64 `json:"progress"`
}

// Log contains raw stderr and stdout outputs
type Log struct {
	Stderr string `json:"stderr"`
	Stdout string `json:"stdout"`
}

// State returns inforation about rsync processing task
func (t Task) State() State {
	return *t.state
}

// Log return structure which contains raw stderr and stdout outputs
func (t Task) Log() Log {
	return Log{
		Stderr: t.log.Stderr,
		Stdout: t.log.Stdout,
	}
}

// Run starts rsync process with options
func (t *Task) Run() error {
	stderr, err := t.rsync.StderrPipe()
	if err != nil {
		return err
	}
	defer stderr.Close()

	stdout, err := t.rsync.StdoutPipe()
	if err != nil {
		return err
	}
	defer stdout.Close()

	go processStdout(t, stdout)
	go processStderr(t, stderr)

	return t.rsync.Run()
}

// NewTask returns new rsync task
func NewTask(source, destination string, rsyncOptions RsyncOptions) *Task {
	// Force set required options
	// rsyncOptions.HumanReadable = true // pb clone change
	// rsyncOptions.Partial = true // pb clone change
	// rsyncOptions.Progress = true // pb clone change
	rsyncOptions.Archive = true

	return &Task{
		rsync: NewRsync(source, destination, rsyncOptions),
		state: &State{},
		log:   &Log{},
	}
}

func processStdout(task *Task, stdout io.Reader) {
	const maxPercents = float64(100)
	const minDivider = 1

	progressMatcher := newMatcher(`\(.+-chk=(\d+.\d+)`)
	speedMatcher := newMatcher(`(\d+\.\d+.{2}\/s)`)

	// Extract data from strings:
	//         999,999 99%  999.99kB/s    0:00:59 (xfr#9, to-chk=999/9999)
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		logStr := scanner.Text()
		if progressMatcher.Match(logStr) {
			task.state.Remain, task.state.Total = getTaskProgress(progressMatcher.Extract(logStr))

			copiedCount := float64(task.state.Total - task.state.Remain)
			task.state.Progress = copiedCount / math.Max(float64(task.state.Total), float64(minDivider)) * maxPercents
		}

		if speedMatcher.Match(logStr) {
			task.state.Speed = getTaskSpeed(speedMatcher.ExtractAllStringSubmatch(logStr, 2))
		}

		task.log.Stdout += logStr + "\n"
	}
}

func processStderr(task *Task, stderr io.Reader) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		task.log.Stderr += scanner.Text() + "\n"
	}
}

func getTaskProgress(remTotalString string) (int, int) {
	const remTotalSeparator = "/"
	const numbersCount = 2
	const (
		indexRem = iota
		indexTotal
	)

	info := strings.Split(remTotalString, remTotalSeparator)
	if len(info) < numbersCount {
		return 0, 0
	}

	remain, _ := strconv.Atoi(info[indexRem])
	total, _ := strconv.Atoi(info[indexTotal])

	return remain, total
}

func getTaskSpeed(data [][]string) string {
	if len(data) < 2 || len(data[1]) < 2 {
		return ""
	}

	return data[1][1]
}
