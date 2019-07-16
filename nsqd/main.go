package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"util"
)

//设置版本号
const VERSION = "0.1.1"

//设置命令行参数
var (
	//版本号
	showVersion = flag.Bool("version", false, "print version string")
	//http连接端口
	webAddress = flag.String("web-address", "0.0.0.0:4151", "<addr>:<port> to listen on for HTTP clients")
	//tcp端口
	tcpAddress = flag.String("tcp-address", "0.0.0.0:4150", "<addr>:<port> to listen on for TCP clients")
	//是否开启debug模式
	debugMode       = flag.Bool("debug", false, "enable debug mode")
	memQueueSize    = flag.Int64("mem-queue-size", 10000, "number of messages to keep in memory (per topic)")
	maxBytesPerFile = flag.Int64("max-bytes-per-file", 1024768, "number of bytes per diskqueue file before rolling")
	//设置cpu相关选项
	cpuProfile = flag.String("cpu-profile", "", "write cpu profile to file")
	goMaxProcs = flag.Int("go-max-procs", 0, "runtime configuration for GOMAXPROCS")
	dataPath   = flag.String("data-path", "", "path to store disk-backed messages")
	//给worker命名
	workerId        = flag.Int64("worker-id", 0, "unique identifier (int) for this worker (will default to a hash of hostname)")
	verbose         = flag.Bool("verbose", false, "enable verbose logging")
	lookupAddresses = util.StringArray{}
)

func init() {
	flag.Var(&lookupAddresses, "lookupd-address", "lookupd address (may be given multiple times)")
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("nsqd v%s\n", VERSION)
		return
	}

	if *goMaxProcs > 0 {
		runtime.GOMAXPROCS(*goMaxProcs)
	}

	if *workerId == 0 {
		//是否设置了worker name，否则根据机器信息生成一个
		hostname, err := os.Hostname()
		if err != nil {
			log.Fatal(err)
		}
		h := md5.New()
		io.WriteString(h, hostname)
		*workerId = int64(crc32.ChecksumIEEE(h.Sum(nil)) % 1024)
	}

	//监听tcp端口
	tcpAddr, err := net.ResolveTCPAddr("tcp", *tcpAddress)
	if err != nil {
		log.Fatal(err)
	}

	//监听http端口
	webAddr, err := net.ResolveTCPAddr("tcp", *webAddress)
	if err != nil {
		log.Fatal(err)
	}

	if *cpuProfile != "" {
		log.Printf("CPU Profiling Enabled")
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	log.Printf("nsqd v%s", VERSION)
	log.Printf("worker id %d", *workerId)

	//退出channel
	exitChan := make(chan int)
	//信号channel
	signalChan := make(chan os.Signal, 1)
	//单独起一个协程来监听信号，收到信号之后就进行进程的退出
	go func() {
		<-signalChan
		exitChan <- 1
	}()
	//监听终端信号
	signal.Notify(signalChan, os.Interrupt)

	nsqd = NewNSQd(*workerId, tcpAddr, webAddr, lookupAddresses, *memQueueSize, *dataPath, *maxBytesPerFile)
	nsqd.Main()
	<-exitChan
	nsqd.Exit()
}
