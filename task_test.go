package grsync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTask(t *testing.T) {
	t.Run("create new empty Task", func(t *testing.T) {
		createdTask := NewTask("a", "b", RsyncOptions{})

		assert.Empty(t, createdTask.Log(), "Task log should return empty string")
		assert.Empty(t, createdTask.State(), "Task should inited with empty state")
	})
}

func TestTaskProgressParse(t *testing.T) {
	progressMatcher := newMatcher(`\(.+-chk=(\d+.\d+)`)
	const taskInfoString = `999,999 99%  999.99kB/s    0:00:59 (xfr#9, to-chk=999/9999)`
	remain, total := getTaskProgress(progressMatcher.Extract(taskInfoString))

	assert.Equal(t, remain, 999)
	assert.Equal(t, total, 9999)
}

func TestTaskProgressWithDifferentChkID(t *testing.T) {
	progressMatcher := newMatcher(`\(.+-chk=(\d+.\d+)`)
	const taskInfoString = `999,999 99%  999.99kB/s    0:00:59 (xfr#9, ir-chk=999/9999)`
	remain, total := getTaskProgress(progressMatcher.Extract(taskInfoString))

	assert.Equal(t, remain, 999)
	assert.Equal(t, total, 9999)
}

func TestTaskSpeedParse(t *testing.T) {
	speedMatcher := newMatcher(`(\d+\.\d+.{2}\/s)`)
	const taskInfoString = `0.00kB/s \n 999,999 99%  999.99kB/s    0:00:59 (xfr#9, ir-chk=999/9999)`
	speed := getTaskSpeed(speedMatcher.ExtractAllStringSubmatch(taskInfoString, 2))
	assert.Equal(t, "999.99kB/s", speed)
}

func TestRunTaskSuccess(t *testing.T) {
	tmpDir := os.TempDir()
	if tmpDir == "" {
		tmpDir = "/tmp"
	}
	tmpDir = filepath.Join(tmpDir, "grsynctest")
	e := os.MkdirAll(tmpDir, os.ModeDir|os.ModePerm)
	assert.Nil(t, e)
	defer os.RemoveAll(tmpDir)
	a := filepath.Join(tmpDir, "a")
	b := filepath.Join(tmpDir, "destDir")
	f, e := os.Create(a)
	assert.Nil(t, e)
	f.Truncate(16 * 1024 * 1024)
	createdTask := NewTask(a, b, RsyncOptions{})
	e = createdTask.Run()
	assert.Nil(t, e)
	_, e = os.Stat(b)
	assert.Nil(t, e)
}

func TestRunTaskFailure(t *testing.T) {
	tmpDir := os.TempDir()
	if tmpDir == "" {
		tmpDir = "/tmp"
	}
	tmpDir = filepath.Join(tmpDir, "grsynctest")
	e := os.MkdirAll(tmpDir, os.ModeDir|os.ModePerm)
	assert.Nil(t, e)
	defer os.RemoveAll(tmpDir)
	a := filepath.Join(tmpDir, "a")
	b := "/nonwritabledir"
	f, e := os.Create(a)
	assert.Nil(t, e)
	f.Truncate(16 * 1024 * 1024)
	createdTask := NewTask(a, b, RsyncOptions{})
	e = createdTask.Run()
	assert.NotNil(t, e)
	_, e = os.Stat(b)
	assert.NotNil(t, e)
}
