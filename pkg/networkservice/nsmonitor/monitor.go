package nsmonitor

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
)

const (
	//netNsMonitorInterval = 250 * time.Millisecond
	netNsMonitorInterval = 1 * time.Second

	monitorResult_Added             = "added to monitoring"
	monitorResult_AlreadyMonitored  = "already monitored"
	monitorResult_UnsupportedScheme = "unsupported scheme"
)

// getProcName returns process name by its pid
func getProcName(pid uint64) (string, error) {
	bytes, err := ioutil.ReadFile(fmt.Sprintf("/proc/%v/stat", pid))
	if err != nil {
		return "", err
	}
	data := string(bytes)
	start := strings.IndexRune(data, '(') + 1
	end := strings.IndexRune(data[start:], ')')
	return data[start : start+end], nil
}

// getInode returns Inode for file
func getInode(file string) (uint64, error) {
	fileinfo, err := os.Stat(file)
	if err != nil {
		return 0, err
	}
	stat, ok := fileinfo.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, errors.New("not a stat_t")
	}
	return stat.Ino, nil
}

type netNsInfo struct {
	pid   uint64
	inode uint64
}

// getAllNetNs returns all network namespace inodes and associated process pids
func getAllNetNs() ([]netNsInfo, error) {
	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		return nil, errors.Wrap(err, "can't read /proc directory")
	}
	var inodes []netNsInfo
	for _, f := range files {
		pid, err := strconv.ParseUint(f.Name(), 10, 64)
		if err != nil {
			continue
		}
		inode, err := getInode(fmt.Sprintf("/proc/%v/ns/net", pid))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			} else {
				return nil, err
			}
		}
		inodes = append(inodes, netNsInfo{
			pid:   pid,
			inode: inode,
		})
	}
	return inodes, nil
}

type monitorItem struct {
	ctx   context.Context
	pause string
}

type netNsMonitor struct {
	mutex sync.Mutex
	nses  map[uint64]monitorItem
	on    bool
}

func newMonitor() *netNsMonitor {
	return &netNsMonitor{
		nses: map[uint64]monitorItem{},
	}
}

func (m *netNsMonitor) AddNsInode(ctx context.Context, nsInodeURL string) (string, error) {
	inodeURL, err := url.Parse(nsInodeURL)
	if err != nil {
		return "", errors.Wrap(err, "invalid url")
	}

	if inodeURL.Scheme != "inode" {
		// We also receive smth like file:///proc/2608554/fd/37
		// it's not an error
		return monitorResult_UnsupportedScheme, nil
	}

	pathParts := strings.Split(inodeURL.Path, "/")
	inode, err := strconv.ParseUint(pathParts[len(pathParts)-1], 10, 64)
	if err != nil {
		return "", errors.Wrap(err, "invalid inode path")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.nses[inode]; ok {
		return monitorResult_AlreadyMonitored, nil
	}

	nses, err := getAllNetNs()
	if err != nil {
		return "", errors.Wrap(err, "unable to get all netns")
	}

	var pausePid uint64 = 0
	for _, ns := range nses {
		if ns.inode == inode {
			proc, err := getProcName(ns.pid)
			if err != nil {
				return "", errors.Wrap(err, "unable to get proc name")
			}
			if proc == "pause" {
				pausePid = ns.pid
			}
		}
	}
	if pausePid == 0 {
		return "", errors.New("pause container not found")
	}

	m.nses[inode] = monitorItem{
		ctx:   ctx,
		pause: fmt.Sprintf("/proc/%v", pausePid),
	}
	if !m.on {
		m.on = true
		go m.monitor()
	}
	return monitorResult_Added, nil
}

func (m *netNsMonitor) monitor() {
	for {
		select {
		case <-time.After(netNsMonitorInterval):
			if !m.checkAllNs() {
				return
			}
		}
	}
}

func (m *netNsMonitor) checkAllNs() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var toDelete []uint64
	for inode, item := range m.nses {
		if _, err := os.Stat(item.pause); err != nil {
			println("netNsMonitor", fmt.Sprintf("%v died", inode))
			toDelete = append(toDelete, inode)
		}
	}

	if len(toDelete) > 0 {
		for _, inode := range toDelete {
			begin.FromContext(m.nses[inode].ctx).Close()
			delete(m.nses, inode)
		}

		if len(m.nses) == 0 {
			m.on = false
			return false
		}
	}

	return true
}
