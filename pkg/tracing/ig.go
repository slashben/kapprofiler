package tracing

import (
	"log"

	tracerexec "github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/exec/tracer"
	tracerexectype "github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/exec/types"
	tracertcp "github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/tcp/tracer"
	tracertcptype "github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/tcp/types"
	eventtypes "github.com/inspektor-gadget/inspektor-gadget/pkg/types"
)

// Global constants
const execTraceName = "trace_exec"

// const openTraceName = "trace_open"
const tcpTraceName = "trace_tcp"

func (t *Tracer) startAppBehaviorTracing() error {

	// Start tracing execve
	err := t.startExecTracing()
	if err != nil {
		log.Printf("error starting exec tracing: %s\n", err)
		return err
	}

	// Start tracing tcp
	err = t.startTcpTracing()
	if err != nil {
		log.Printf("error starting tcp tracing: %s\n", err)
		return err
	}

	return nil
}

func (t *Tracer) execEventCallback(event *tracerexectype.Event) {
	if event.Type == eventtypes.NORMAL && event.Retval > -1 {
		execveEvent := &ExecveEvent{
			ContainerID: event.Container,
			PodName:     event.Pod,
			Namespace:   event.Namespace,
			PathName:    event.Args[0],
			Args:        event.Args[1:],
			Env:         []string{},
			Timestamp:   int64(event.Timestamp),
		}
		t.eventSink.SendExecveEvent(execveEvent)
	}
}

func (t *Tracer) startExecTracing() error {
	// Add exec tracer
	if err := t.tCollection.AddTracer(execTraceName, t.containerSelector); err != nil {
		log.Printf("error adding tracer: %s\n", err)
		return err
	}

	// Get mount namespace map to filter by containers
	execMountnsmap, err := t.tCollection.TracerMountNsMap(execTraceName)
	if err != nil {
		log.Printf("failed to get execMountnsmap: %s\n", err)
		return err
	}

	// Create the exec tracer
	tracerExec, err := tracerexec.NewTracer(&tracerexec.Config{MountnsMap: execMountnsmap}, t.cCollection, t.execEventCallback)
	if err != nil {
		log.Printf("error creating tracer: %s\n", err)
		return err
	}
	t.execTracer = tracerExec
	return nil
}

func (t *Tracer) tcpEventCallback(event *tracertcptype.Event) {
	if event.Type == eventtypes.NORMAL {
		tcpEvent := &TcpEvent{
			ContainerID: event.Container,
			PodName:     event.Pod,
			Namespace:   event.Namespace,
			Source:      event.Saddr,
			SourcePort:  int(event.Sport),
			Destination: event.Daddr,
			DestPort:    int(event.Dport),
			Operation:   event.Operation,
			Timestamp:   int64(event.Timestamp),
		}
		t.eventSink.SendTcpEvent(tcpEvent)
	}
}

func (t *Tracer) startTcpTracing() error {
	// Add tcp tracer
	if err := t.tCollection.AddTracer(tcpTraceName, t.containerSelector); err != nil {
		log.Printf("error adding tcp tracer: %s\n", err)
		return err
	}

	// Get mount namespace map to filter by containers
	tcpMountnsmap, err := t.tCollection.TracerMountNsMap(tcpTraceName)
	if err != nil {
		log.Printf("failed to get tcpMountnsmap: %s\n", err)
		return err
	}

	// Create the tcp tracer
	tracerTcp, err := tracertcp.NewTracer(&tracertcp.Config{MountnsMap: tcpMountnsmap}, t.cCollection, t.tcpEventCallback)
	if err != nil {
		log.Printf("error creating tracer: %s\n", err)
		return err
	}
	t.tcpTracer = tracerTcp
	return nil
}

func (t *Tracer) stopAppBehaviorTracing() error {
	var err error
	err = nil
	// Stop exec tracer
	if err = t.stopExecTracing(); err != nil {
		log.Printf("error stopping exec tracing: %s\n", err)
	}
	// Stop tcp tracer
	if err = t.stopTcpTracing(); err != nil {
		log.Printf("error stopping tcp tracing: %s\n", err)
	}
	return err
}

func (t *Tracer) stopExecTracing() error {
	// Stop exec tracer
	if err := t.tCollection.RemoveTracer(execTraceName); err != nil {
		log.Printf("error removing tracer: %s\n", err)
		return err
	}
	t.execTracer.Stop()
	return nil
}

func (t *Tracer) stopTcpTracing() error {
	// Stop tcp tracer
	if err := t.tCollection.RemoveTracer(tcpTraceName); err != nil {
		log.Printf("error removing tracer: %s\n", err)
		return err
	}
	t.tcpTracer.Stop()
	return nil
}
