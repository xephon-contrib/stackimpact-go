package internal

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

const AgentVersion = "1.2.2"
const SAASDashboardAddress = "https://agent-api.stackimpact.com"

var agentStarted bool = false

type Agent struct {
	nextId             int64
	runId              string
	runTs              int64
	overheadLock       *sync.Mutex
	apiRequest         *APIRequest
	configLoader       *ConfigLoader
	messageQueue       *MessageQueue
	processReporter    *ProcessReporter
	cpuReporter        *CPUReporter
	allocationReporter *AllocationReporter
	blockReporter      *BlockReporter
	segmentReporter    *SegmentReporter
	errorReporter      *ErrorReporter

	// Options
	DashboardAddress string
	AgentKey         string
	AppName          string
	AppVersion       string
	AppEnvironment   string
	HostName         string
	Debug            bool
	disableProfiling bool
}

func NewAgent() *Agent {
	a := &Agent{
		nextId:             0,
		runId:              "",
		runTs:              time.Now().Unix(),
		overheadLock:       &sync.Mutex{},
		apiRequest:         nil,
		configLoader:       nil,
		messageQueue:       nil,
		processReporter:    nil,
		cpuReporter:        nil,
		allocationReporter: nil,
		blockReporter:      nil,
		segmentReporter:    nil,
		errorReporter:      nil,

		DashboardAddress: SAASDashboardAddress,
		AgentKey:         "",
		AppName:          "",
		AppVersion:       "",
		AppEnvironment:   "",
		HostName:         "",
		Debug:            false,
		disableProfiling: false,
	}

	a.runId = a.uuid()

	a.apiRequest = newAPIRequest(a)
	a.configLoader = newConfigLoader(a)
	a.messageQueue = newMessageQueue(a)
	a.processReporter = newProcessReporter(a)
	a.cpuReporter = newCPUReporter(a)
	a.allocationReporter = newAllocationReporter(a)
	a.blockReporter = newBlockReporter(a)
	a.segmentReporter = newSegmentReporter(a)
	a.errorReporter = newErrorReporter(a)

	return a
}

func (a *Agent) Start() {
	if agentStarted {
		a.log("Agent configuration failed. Another agent has already been initialized.")
		return
	}
	agentStarted = true

	if a.HostName == "" {
		hostName, err := os.Hostname()
		if err != nil {
			a.error(err)
		}
		a.HostName = hostName
	}

	a.configLoader.start()
	a.messageQueue.start()
	a.processReporter.start()
	a.cpuReporter.start()
	a.allocationReporter.start()
	a.blockReporter.start()
	a.segmentReporter.start()
	a.errorReporter.start()

	a.log("Agent started.")

	return
}

func (a *Agent) RecordSegment(path []string, duration int64) {
	if !agentStarted {
		return
	}

	a.segmentReporter.recordSegment(path, duration)
}

func (a *Agent) RecordError(group string, msg interface{}, skipFrames int) {
	if !agentStarted {
		return
	}

	var err error
	switch v := msg.(type) {
	case error:
		err = v
	default:
		err = fmt.Errorf("%v", v)
	}

	a.errorReporter.recordError(group, err, skipFrames+1)
}

func (a *Agent) log(format string, values ...interface{}) {
	if a.Debug {
		fmt.Printf("["+time.Now().Format(time.StampMilli)+"]"+
			" StackImpact "+AgentVersion+": "+
			format+"\n", values...)
	}
}

func (a *Agent) error(err error) {
	if a.Debug {
		fmt.Println("[" + time.Now().Format(time.StampMilli) + "]" +
			" StackImpact " + AgentVersion + ": Error")
		fmt.Println(err)
	}
}

func (a *Agent) recoverAndLog() {
	if err := recover(); err != nil {
		a.log("Recovered from panic in agent: %v", err)
	}
}

func (a *Agent) uuid() string {
	a.nextId++

	uuid :=
		strconv.FormatInt(time.Now().Unix(), 10) +
			strconv.Itoa(rand.Intn(1000000000)) +
			strconv.FormatInt(a.nextId, 10)

	return sha1String(uuid)
}

func sha1String(s string) string {
	sha1 := sha1.New()
	sha1.Write([]byte(s))

	return hex.EncodeToString(sha1.Sum(nil))
}
