package main

import (
	"github.com/bmizerany/assert"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestDiskQueue(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	dqName := "test_disk_queue" + strconv.Itoa(int(time.Now().Unix()))
	dq := NewDiskQueue(dqName, os.TempDir(), 1024)
	assert.NotEqual(t, dq, nil)
	assert.Equal(t, dq.Depth(), int64(0))

	msg := []byte("test")
	err := dq.Put(msg)
	assert.Equal(t, err, nil)
	assert.Equal(t, dq.Depth(), int64(1))

	msgOut := <-dq.ReadChan()
	assert.Equal(t, msgOut, msg)
}

func TestDiskQueueRoll(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	dqName := "test_disk_queue_roll" + strconv.Itoa(int(time.Now().Unix()))
	dq := NewDiskQueue(dqName, os.TempDir(), 100)
	assert.NotEqual(t, dq, nil)
	assert.Equal(t, dq.Depth(), int64(0))

	msg := []byte("aaaaaaaaaa")
	for i := 0; i < 10; i++ {
		err := dq.Put(msg)
		assert.Equal(t, err, nil)
		assert.Equal(t, dq.Depth(), int64(i+1))
	}

	assert.Equal(t, dq.(*DiskQueue).writeFileNum, int64(1))
	assert.Equal(t, dq.(*DiskQueue).writePos, int64(28))
}

func TestDiskQueueEmpty(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	dqName := "test_disk_queue_empty" + strconv.Itoa(int(time.Now().Unix()))
	dq := NewDiskQueue(dqName, os.TempDir(), 100)
	assert.NotEqual(t, dq, nil)
	assert.Equal(t, dq.Depth(), int64(0))

	msg := []byte("aaaaaaaaaa")
	for i := 0; i < 100; i++ {
		err := dq.Put(msg)
		assert.Equal(t, err, nil)
		assert.Equal(t, dq.Depth(), int64(i+1))
	}

	dq.Empty()
	assert.Equal(t, dq.Depth(), int64(0))
	assert.Equal(t, dq.(*DiskQueue).readFileNum, dq.(*DiskQueue).writeFileNum)
	assert.Equal(t, dq.(*DiskQueue).readPos, dq.(*DiskQueue).writePos)
	assert.Equal(t, dq.(*DiskQueue).nextReadPos, dq.(*DiskQueue).readPos)
}

func BenchmarkDiskQueuePut(b *testing.B) {
	b.StopTimer()
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)
	dqName := "bench_disk_queue_put" + strconv.Itoa(b.N) + strconv.Itoa(int(time.Now().Unix()))
	dq := NewDiskQueue(dqName, os.TempDir(), 1024)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		dq.Put([]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	}
}

// this benchmark should be run via:
//    $ go test -test.bench 'DiskQueueGet' -test.benchtime 0.1
// (so that it does not perform too many iterations)
func BenchmarkDiskQueueGet(b *testing.B) {
	b.StopTimer()
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)
	dqName := "bench_disk_queue_get" + strconv.Itoa(b.N) + strconv.Itoa(int(time.Now().Unix()))
	dq := NewDiskQueue(dqName, os.TempDir(), 1024768)
	for i := 0; i < b.N; i++ {
		dq.Put([]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		<-dq.ReadChan()
	}
}
