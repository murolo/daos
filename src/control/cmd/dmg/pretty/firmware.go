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

// PrintSCMFirmwareQueryMap formats the firmware query results for human readability.
func PrintSCMFirmwareQueryMap(fwMap control.HostSCMQueryMap, out io.Writer,
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

			if res.Info == nil {
				fmt.Fprintf(iw1, "%s: No information available\n", errorPrefix)
				continue
			}

			fmt.Fprintf(iw1, "Active Version: %s\n", getPrintVersion(res.Info.ActiveVersion))
			fmt.Fprintf(iw1, "Staged Version: %s\n", getPrintVersion(res.Info.StagedVersion))
			fmt.Fprintf(iw1, "Maximum Firmware Image Size: %s\n", humanize.IBytes(uint64(res.Info.ImageMaxSizeBytes)))
			fmt.Fprintf(iw1, "Last Update Status: %s\n", res.Info.UpdateStatus)
		}
	}

	return w.Err
}

// PrintSCMFirmwareUpdateMapVerbose formats the firmware query results in a
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

type hostDeviceSet struct {
	hosts   *hostlist.HostSet
	devices []string
}

func (h hostDeviceSet) addHost(host string) {
	h.hosts.Insert(host)
}

func (h hostDeviceSet) addDevice(device string) {
	h.devices = append(h.devices, device)
}

func newHostDeviceSet() (*hostDeviceSet, error) {
	h := &hostDeviceSet{}
	hosts, err := hostlist.CreateSet("")
	if err != nil {
		return nil, err
	}
	h.hosts = hosts
	h.devices = make([]string, 0)
	return h, nil
}

type hostDeviceResultMap map[string]*hostDeviceSet

func (m hostDeviceResultMap) addResult(resultStr string) error {
	if _, ok := m[resultStr]; !ok {
		newSet, err := newHostDeviceSet()
		if err != nil {
			return err
		}
		m[resultStr] = *newSet
	}
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
			err := condensed.addResult(scmNotFound)
			if err != nil {
				return nil, err
			}
			condensed[scmNotFound].addHost(host)
			continue
		}

		for _, devRes := range results {
			resultStr := scmUpdateSuccess
			if devRes.Error != nil {
				resultStr = fmt.Sprintf("%s: %s\n", errorPrefix, devRes.Error.Error())
			}

			err := condensed.addResult(resultStr)
			if err != nil {
				return nil, err
			}
			condensed[resultStr].addHost(host)
			condensed[resultStr].addDevice(devRes.Module.String())
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
		hosts := control.GetPrintHosts(set.hosts.RangedString(), opts...)
		lineBreak := strings.Repeat("-", len(hosts))
		fmt.Fprintf(out, "%s\n%s\n%s\n", lineBreak, hosts, lineBreak)

		iw := txtfmt.NewIndentWriter(out)
		fmt.Fprintln(iw, result)

		iw2 := txtfmt.NewIndentWriter(iw)
		for _, dev := range set.devices {
			fmt.Fprintln(iw2, dev)
		}
	}

	return w.Err
}
