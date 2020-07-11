//
// (C) Copyright 2020 Intel Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// GOVERNMENT LICENSE RIGHTS-OPEN SOURCE SOFTWARE
// The Government's rights to use, modify, reproduce, release, perform, display,
// or disclose this software are subject to the terms of the Apache License as
// provided in Contract No. 8F-30005.
// Any reproduction of computer software, computer software documentation, or
// portions thereof marked with this legend must also reproduce the markings.
//
// +build firmware

package pretty

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"

	"github.com/daos-stack/daos/src/control/lib/control"
	"github.com/daos-stack/daos/src/control/lib/hostlist"
	"github.com/daos-stack/daos/src/control/lib/txtfmt"
	"github.com/daos-stack/daos/src/control/server/storage"
)

const (
	scmUpdateSuccess = "Success - The new firmware was staged. A reboot is required to apply."
	scmNotFound      = "No SCM devices detected"
	errorPrefix      = "Error"
)

func getPrintVersion(version string) string {
	if version == "" {
		return "N/A"
	}
	return version
}

// PrintSCMFirmwareQueryMapVerbose formats the firmware query results in a detailed format.
func PrintSCMFirmwareQueryMapVerbose(fwMap control.HostSCMQueryMap, out io.Writer,
	opts ...control.PrintConfigOption) error {
	w := txtfmt.NewErrWriter(out)

	for _, host := range fwMap.Keys() {
		fwResults := fwMap[host]
		lineBreak := strings.Repeat("-", len(host))
		fmt.Fprintf(out, "%s\n%s\n%s\n", lineBreak, host, lineBreak)

		iw := txtfmt.NewIndentWriter(out)
		if len(fwResults) == 0 {
			fmt.Fprintln(iw, scmNotFound)
			continue
		}

		for _, res := range fwResults {
			err := printScmModule(&res.Module, iw)
			if err != nil {
				return err
			}

			iw1 := txtfmt.NewIndentWriter(iw)

			if res.Error != nil {
				fmt.Fprintf(iw1, "%s: %s\n", errorPrefix, res.Error.Error())
				continue
			}

			printSCMFirmwareInfo(res.Info, iw1)
		}
	}

	return w.Err
}

func printSCMFirmwareInfo(info *storage.ScmFirmwareInfo, out io.Writer) {
	if info == nil {
		fmt.Fprintf(out, "%s: No information available\n", errorPrefix)
		return
	}

	fmt.Fprintf(out, "Active Version: %s\n", getPrintVersion(info.ActiveVersion))
	fmt.Fprintf(out, "Staged Version: %s\n", getPrintVersion(info.StagedVersion))
	fmt.Fprintf(out, "Maximum Firmware Image Size: %s\n", humanize.IBytes(uint64(info.ImageMaxSizeBytes)))
	fmt.Fprintf(out, "Last Update Status: %s\n", info.UpdateStatus)
}

func condenseSCMQueryMap(fwMap control.HostSCMQueryMap) (hostDeviceResultMap, error) {
	condensed := make(hostDeviceResultMap)
	for _, host := range fwMap.Keys() {
		results := fwMap[host]
		if len(results) == 0 {
			err := condensed.AddHost(scmNotFound, host)
			if err != nil {
				return nil, err
			}
			continue
		}

		for _, devRes := range results {
			var resultStr string
			if devRes.Error != nil {
				resultStr = fmt.Sprintf("%s: %s", errorPrefix, devRes.Error.Error())
			} else {
				var b strings.Builder
				printSCMFirmwareInfo(devRes.Info, &b)
				resultStr = b.String()
			}

			err := condensed.AddHostDevice(resultStr, host, devRes.Module.String())
			if err != nil {
				return nil, err
			}
		}
	}
	return condensed, nil
}

// PrintSCMFirmwareQueryMap formats the firmware query results in a condensed format.
func PrintSCMFirmwareQueryMap(fwMap control.HostSCMQueryMap, out io.Writer,
	opts ...control.PrintConfigOption) error {
	condensed, err := condenseSCMQueryMap(fwMap)
	if err != nil {
		return err
	}

	w := txtfmt.NewErrWriter(out)
	for _, result := range condensed.Keys() {
		set, ok := condensed[result]
		if !ok {
			continue
		}
		hosts := control.GetPrintHosts(set.Hosts.RangedString(), opts...)
		lineBreak := strings.Repeat("-", len(hosts))
		fmt.Fprintf(out, "%s\n%s\n%s\n", lineBreak, hosts, lineBreak)

		iw := txtfmt.NewIndentWriter(out)
		fmt.Fprintln(iw, result)

		iw2 := txtfmt.NewIndentWriter(iw)
		for _, dev := range set.Devices {
			fmt.Fprintln(iw2, dev)
		}
	}

	return w.Err
}

// PrintSCMFirmwareUpdateMapVerbose formats the firmware update results in a
// detailed format.
func PrintSCMFirmwareUpdateMapVerbose(fwMap control.HostSCMUpdateMap, out io.Writer,
	opts ...control.PrintConfigOption) error {
	w := txtfmt.NewErrWriter(out)

	for _, host := range fwMap.Keys() {
		fwResults := fwMap[host]
		lineBreak := strings.Repeat("-", len(host))
		fmt.Fprintf(out, "%s\n%s\n%s\n", lineBreak, host, lineBreak)

		iw := txtfmt.NewIndentWriter(out)
		if len(fwResults) == 0 {
			fmt.Fprintln(iw, scmNotFound)
			continue
		}

		for _, res := range fwResults {
			err := printScmModule(&res.Module, iw)
			if err != nil {
				return err
			}

			iw1 := txtfmt.NewIndentWriter(iw)

			if res.Error != nil {
				fmt.Fprintf(iw1, "%s: %s\n", errorPrefix, res.Error.Error())
				continue
			}

			fmt.Fprintf(iw1, "%s\n", scmUpdateSuccess)
		}
	}

	return w.Err
}

type HostDeviceSet struct {
	Hosts   *hostlist.HostSet
	Devices []string
}

func (h *HostDeviceSet) AddHost(host string) {
	h.Hosts.Insert(host)
}

func (h *HostDeviceSet) AddDevice(device string) {
	h.Devices = append(h.Devices, device)
}

func newHostDeviceSet() (*HostDeviceSet, error) {
	hosts, err := hostlist.CreateSet("")
	if err != nil {
		return nil, err
	}
	return &HostDeviceSet{
		Hosts: hosts,
	}, nil
}

type hostDeviceResultMap map[string]*HostDeviceSet

func (m hostDeviceResultMap) AddHostDevice(resultStr string, host string, device string) error {
	err := m.AddHost(resultStr, host)
	if err != nil {
		return err
	}
	m[resultStr].AddDevice(device)
	return nil
}

func (m hostDeviceResultMap) AddHost(resultStr string, host string) error {
	if _, ok := m[resultStr]; !ok {
		newSet, err := newHostDeviceSet()
		if err != nil {
			return err
		}
		m[resultStr] = newSet
	}

	m[resultStr].AddHost(host)
	return nil
}

func (m hostDeviceResultMap) Keys() []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func condenseSCMUpdateMap(fwMap control.HostSCMUpdateMap) (hostDeviceResultMap, error) {
	condensed := make(hostDeviceResultMap)
	for _, host := range fwMap.Keys() {
		results := fwMap[host]
		if len(results) == 0 {
			err := condensed.AddHost(scmNotFound, host)
			if err != nil {
				return nil, err
			}
			continue
		}

		for _, devRes := range results {
			resultStr := scmUpdateSuccess
			if devRes.Error != nil {
				resultStr = fmt.Sprintf("%s: %s", errorPrefix, devRes.Error.Error())
			}

			err := condensed.AddHostDevice(resultStr, host, devRes.Module.String())
			if err != nil {
				return nil, err
			}
		}
	}
	return condensed, nil
}

// PrintSCMFirmwareUpdateMap prints the update results in a condensed format.
func PrintSCMFirmwareUpdateMap(fwMap control.HostSCMUpdateMap, out io.Writer,
	opts ...control.PrintConfigOption) error {
	condensed, err := condenseSCMUpdateMap(fwMap)
	if err != nil {
		return err
	}

	w := txtfmt.NewErrWriter(out)

	for _, result := range condensed.Keys() {
		set, ok := condensed[result]
		if !ok {
			continue
		}
		hosts := control.GetPrintHosts(set.Hosts.RangedString(), opts...)
		lineBreak := strings.Repeat("-", len(hosts))
		fmt.Fprintf(out, "%s\n%s\n%s\n", lineBreak, hosts, lineBreak)

		iw := txtfmt.NewIndentWriter(out)
		fmt.Fprintln(iw, result)

		iw2 := txtfmt.NewIndentWriter(iw)
		for _, dev := range set.Devices {
			fmt.Fprintln(iw2, dev)
		}
	}

	return w.Err
}
