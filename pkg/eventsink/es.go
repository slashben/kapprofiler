package eventsink

import (
	"fmt"
	"kapprofiler/pkg/tracing"
	"os"

	"log"

	bolt "go.etcd.io/bbolt"
)

type EventSink struct {
	homeDir                  string
	fileDB                   *bolt.DB
	execveEventChannel       chan *tracing.ExecveEvent
	tcpEventChannel          chan *tracing.TcpEvent
	openEventChannel         chan *tracing.OpenEvent
	capabilitiesEventChannel chan *tracing.CapabilitiesEvent
	dnsEventChannel          chan *tracing.DnsEvent
}

func NewEventSink(homeDir string) (*EventSink, error) {
	return &EventSink{homeDir: homeDir}, nil
}

func (es *EventSink) Start() error {
	// Setup bolt database
	if es.homeDir == "" {
		// TODO: Use a better default
		es.homeDir = "/tmp"
	}
	db, err := bolt.Open(es.homeDir+"/execve-events.db", 0600, nil)
	if err != nil {
		return err
	}
	es.fileDB = db

	// Create the channel for execve events
	es.execveEventChannel = make(chan *tracing.ExecveEvent, 10000)

	// Create the channel for tcp events
	es.tcpEventChannel = make(chan *tracing.TcpEvent, 10000)

	// Create the channel for the open events
	es.openEventChannel = make(chan *tracing.OpenEvent, 10000)

	// Create the channel for the capabilities events
	es.capabilitiesEventChannel = make(chan *tracing.CapabilitiesEvent, 10000)

	// Create the channel for the dns events
	es.dnsEventChannel = make(chan *tracing.DnsEvent, 10000)

	// Start the execve event worker
	go es.execveEventWorker()

	// Start the tcp event worker
	go es.tcpEventWorker()

	// Start the open event worker
	go es.openEventWorker()

	// Start the capabilities event worker
	go es.capabilitiesEventWorker()

	// Start the dns event worker
	go es.dnsEventWorker()

	return nil
}

func (es *EventSink) Stop() error {
	// Close the channel for execve events
	close(es.execveEventChannel)

	// Close the channel for tcp events
	close(es.tcpEventChannel)

	// Close the channel for open events
	close(es.openEventChannel)

	// Close the channel for capabilities events
	close(es.capabilitiesEventChannel)

	// Close the channel for dns events
	close(es.dnsEventChannel)

	// Close the bolt database
	err := es.fileDB.Close()
	if err != nil {
		return err
	}

	// Delete boltdb file
	os.Remove(es.homeDir + "/execve-events.db")

	return nil
}

func (es *EventSink) dnsEventWorker() error {
	for event := range es.capabilitiesEventChannel {
		bucket := fmt.Sprintf("dns-%s-%s-%s", event.Namespace, event.PodName, event.ContainerID)
		err := es.fileDB.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				log.Printf("error creating bucket: %s\n", err)
				return err
			}
			sEvent, err := event.GobEncode()
			if err != nil {
				log.Printf("error encoding dns event: %s\n", err)
				return err
			}
			err = b.Put(sEvent, nil)
			if err != nil {
				log.Printf("error storing dns event: %s\n", err)
				return err
			}
			return nil
		})
		if err != nil {
			log.Printf("error storing dns event: %s\n", err)
		}
	}

	return nil
}

func (es *EventSink) capabilitiesEventWorker() error {
	for event := range es.capabilitiesEventChannel {
		bucket := fmt.Sprintf("capabilities-%s-%s-%s", event.Namespace, event.PodName, event.ContainerID)
		err := es.fileDB.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				log.Printf("error creating bucket: %s\n", err)
				return err
			}
			sEvent, err := event.GobEncode()
			if err != nil {
				log.Printf("error encoding capabilities event: %s\n", err)
				return err
			}
			err = b.Put(sEvent, nil)
			if err != nil {
				log.Printf("error storing capabilities event: %s\n", err)
				return err
			}
			return nil
		})
		if err != nil {
			log.Printf("error storing capabilities event: %s\n", err)
		}
	}

	return nil
}

func (es *EventSink) openEventWorker() error {
	for event := range es.openEventChannel {
		bucket := fmt.Sprintf("open-%s-%s-%s", event.Namespace, event.PodName, event.ContainerID)
		err := es.fileDB.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				log.Printf("error creating bucket: %s\n", err)
				return err
			}
			sEvent, err := event.GobEncode()
			if err != nil {
				log.Printf("error encoding open event: %s\n", err)
				return err
			}
			err = b.Put(sEvent, nil)
			if err != nil {
				log.Printf("error storing open event: %s\n", err)
				return err
			}
			return nil
		})
		if err != nil {
			log.Printf("error storing open event: %s\n", err)
		}
	}

	return nil
}

func (es *EventSink) execveEventWorker() error {
	// TODO: Implement this with batch writes

	// Wait for execve events and store them in the database
	for event := range es.execveEventChannel {
		bucket := fmt.Sprintf("execve-%s-%s-%s", event.Namespace, event.PodName, event.ContainerID)
		err := es.fileDB.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				log.Printf("error creating bucket: %s\n", err)
				return err
			}
			sEvent, err := event.GobEncode()
			if err != nil {
				log.Printf("error encoding execve event: %s\n", err)
				return err
			}
			err = b.Put(sEvent, nil)
			if err != nil {
				log.Printf("error storing execve event: %s\n", err)
				return err
			}
			return nil
		})
		if err != nil {
			log.Printf("error storing execve event: %s\n", err)
		}
	}

	return nil
}

func (es *EventSink) tcpEventWorker() error {
	// Wait for tcp events and store them in the database
	for event := range es.tcpEventChannel {
		bucket := fmt.Sprintf("tcp-%s-%s-%s", event.Namespace, event.PodName, event.ContainerID)
		err := es.fileDB.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				log.Printf("error creating bucket: %s\n", err)
				return err
			}
			sEvent, err := event.GobEncode()
			if err != nil {
				log.Printf("error encoding tcp event: %s\n", err)
				return err
			}
			err = b.Put(sEvent, nil)
			if err != nil {
				log.Printf("error storing tcp event: %s\n", err)
				return err
			}
			return nil
		})
		if err != nil {
			log.Printf("error storing tcp event: %s\n", err)
		}
	}
	return nil
}

func (es *EventSink) CleanupContainer(namespace string, podName string, containerID string) error {
	bucket := fmt.Sprintf("execve-%s-%s-%s", namespace, podName, containerID)
	err := es.fileDB.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(bucket))
		if err != nil {
			return err
		}
		return nil
	})
	bucket = fmt.Sprintf("tcp-%s-%s-%s", namespace, podName, containerID)
	err = es.fileDB.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(bucket))
		if err != nil {
			return err
		}
		return nil
	})
	bucket = fmt.Sprintf("open-%s-%s-%s", namespace, podName, containerID)
	err = es.fileDB.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(bucket))
		if err != nil {
			return err
		}
		return nil
	})

	bucket = fmt.Sprintf("capabilities-%s-%s-%s", namespace, podName, containerID)
	err = es.fileDB.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(bucket))
		if err != nil {
			return err
		}
		return nil
	})

	bucket = fmt.Sprintf("dns-%s-%s-%s", namespace, podName, containerID)
	err = es.fileDB.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(bucket))
		if err != nil {
			return err
		}
		return nil
	})

	return err
}

func (es *EventSink) GetDnsEvents(namespace string, podName string, containerID string) ([]*tracing.DnsEvent, error) {
	bucket := fmt.Sprintf("dns-%s-%s-%s", namespace, podName, containerID)
	var events []*tracing.DnsEvent
	err := es.fileDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		b.ForEach(func(k, v []byte) error {
			event := &tracing.DnsEvent{}
			err := event.GobDecode(k)
			if err != nil {
				return err
			}
			events = append(events, event)
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (es *EventSink) GetCapabilitiesEvents(namespace string, podName string, containerID string) ([]*tracing.CapabilitiesEvent, error) {
	bucket := fmt.Sprintf("capabilities-%s-%s-%s", namespace, podName, containerID)
	var events []*tracing.CapabilitiesEvent
	err := es.fileDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		b.ForEach(func(k, v []byte) error {
			event := &tracing.CapabilitiesEvent{}
			err := event.GobDecode(k)
			if err != nil {
				return err
			}
			events = append(events, event)
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (es *EventSink) GetExecveEvents(namespace string, podName string, containerID string) ([]*tracing.ExecveEvent, error) {
	bucket := fmt.Sprintf("execve-%s-%s-%s", namespace, podName, containerID)
	var events []*tracing.ExecveEvent
	err := es.fileDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		b.ForEach(func(k, v []byte) error {
			event := &tracing.ExecveEvent{}
			err := event.GobDecode(k)
			if err != nil {
				return err
			}
			events = append(events, event)
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (es *EventSink) GetTcpEvents(namespace string, podName string, containerID string) ([]*tracing.TcpEvent, error) {
	bucket := fmt.Sprintf("tcp-%s-%s-%s", namespace, podName, containerID)
	var events []*tracing.TcpEvent
	err := es.fileDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		b.ForEach(func(k, v []byte) error {
			event := &tracing.TcpEvent{}
			err := event.GobDecode(k)
			if err != nil {
				return err
			}
			events = append(events, event)
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (es *EventSink) GetOpenEvents(namespace string, podName string, containerID string) ([]*tracing.OpenEvent, error) {
	bucket := fmt.Sprintf("open-%s-%s-%s", namespace, podName, containerID)
	var events []*tracing.OpenEvent
	err := es.fileDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		b.ForEach(func(k, v []byte) error {
			event := &tracing.OpenEvent{}
			err := event.GobDecode(k)
			if err != nil {
				return err
			}
			events = append(events, event)
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (es *EventSink) SendExecveEvent(event *tracing.ExecveEvent) {
	es.execveEventChannel <- event
}

func (es *EventSink) SendTcpEvent(event *tracing.TcpEvent) {
	es.tcpEventChannel <- event
}

func (es *EventSink) SendOpenEvent(event *tracing.OpenEvent) {
	es.openEventChannel <- event
}

func (es *EventSink) SendCapabilitiesEvent(event *tracing.CapabilitiesEvent) {
	es.capabilitiesEventChannel <- event
}

func (es *EventSink) SendDnsEvent(event *tracing.DnsEvent) {
	es.dnsEventChannel <- event
}

func (es *EventSink) Close() error {
	return es.fileDB.Close()
}
