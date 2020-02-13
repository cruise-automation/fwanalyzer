/*
Copyright 2019-present, Cruise LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package capability

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

/*
 * Consts and structs are based on the linux kernel headers for capabilities
 * see: https://github.com/torvalds/linux/blob/master/include/uapi/linux/capability.h
 */

const (
	CAP_CHOWN            = 0
	CAP_DAC_OVERRIDE     = 1
	CAP_DAC_READ_SEARCH  = 2
	CAP_FOWNER           = 3
	CAP_FSETID           = 4
	CAP_KILL             = 5
	CAP_SETGID           = 6
	CAP_SETUID           = 7
	CAP_SETPCAP          = 8
	CAP_LINUX_IMMUTABLE  = 9
	CAP_NET_BIND_SERVICE = 10
	CAP_NET_BROADCAST    = 11
	CAP_NET_ADMIN        = 12
	CAP_NET_RAW          = 13
	CAP_IPC_LOCK         = 14
	CAP_IPC_OWNER        = 15
	CAP_SYS_MODULE       = 16
	CAP_SYS_RAWIO        = 17
	CAP_SYS_CHROOT       = 18
	CAP_SYS_PTRACE       = 19
	CAP_SYS_PACCT        = 20
	CAP_SYS_ADMIN        = 21
	CAP_SYS_BOOT         = 22
	CAP_SYS_NICE         = 23
	CAP_SYS_RESOURCE     = 24
	CAP_SYS_TIME         = 25
	CAP_SYS_TTY_CONFIG   = 26
	CAP_MKNOD            = 27
	CAP_LEASE            = 28
	CAP_AUDIT_WRITE      = 29
	CAP_AUDIT_CONTROL    = 30
	CAP_SETFCAP          = 31
	CAP_MAC_OVERRIDE     = 32
	CAP_MAC_ADMIN        = 33
	CAP_SYSLOG           = 34
	CAP_WAKE_ALARM       = 35
	CAP_BLOCK_SUSPEND    = 36
	CAP_AUDIT_READ       = 37
	CAP_LAST_CAP         = CAP_AUDIT_READ
)

var CapabilityNames = []string{
	"CAP_CHOWN",
	"CAP_DAC_OVERRIDE",
	"CAP_DAC_READ_SEARCH",
	"CAP_FOWNER",
	"CAP_FSETID",
	"CAP_KILL",
	"CAP_SETGID",
	"CAP_SETUID",
	"CAP_SETPCAP",
	"CAP_LINUX_IMMUTABLE",
	"CAP_NET_BIND_SERVICE",
	"CAP_NET_BROADCAST",
	"CAP_NET_ADMIN",
	"CAP_NET_RAW",
	"CAP_IPC_LOCK",
	"CAP_IPC_OWNER",
	"CAP_SYS_MODULE",
	"CAP_SYS_RAWIO",
	"CAP_SYS_CHROOT",
	"CAP_SYS_PTRACE",
	"CAP_SYS_PACCT",
	"CAP_SYS_ADMIN",
	"CAP_SYS_BOOT",
	"CAP_SYS_NICE",
	"CAP_SYS_RESOURCE",
	"CAP_SYS_TIME",
	"CAP_SYS_TTY_CONFIG",
	"CAP_MKNOD",
	"CAP_LEASE",
	"CAP_AUDIT_WRITE",
	"CAP_AUDIT_CONTROL",
	"CAP_SETFCAP",
	"CAP_MAC_OVERRIDE",
	"CAP_MAC_ADMIN",
	"CAP_SYSLOG",
	"CAP_WAKE_ALARM",
	"CAP_BLOCK_SUSPEND",
	"CAP_AUDIT_READ"}

const capOffset = 2
const CapByteSizeMax = 24

const (
	CAP_PERMITTED   = 0
	CAP_INHERITABLE = 1
)

/*
 * capabilities are store in the vfs_cap_data struct
 *

struct vfs_cap_data {
	__le32 magic_etc;            // Little endian
	struct {
		__le32 permitted;    // Little endian
		__le32 inheritable;  // Little endian
	} data[VFS_CAP_U32];
};
*/

// https://github.com/torvalds/linux/blob/master/include/uapi/linux/capability.h#L373
func capValid(cap uint32) bool {
	// cap >= 0 && cap <= CAP_LAST_CAP
	return cap <= CAP_LAST_CAP
}

// https://github.com/torvalds/linux/blob/master/include/uapi/linux/capability.h#L379
func capIndex(cap uint32) int {
	return int(cap>>5) * capOffset
}

// https://github.com/torvalds/linux/blob/master/include/uapi/linux/capability.h#L380
func capMask(cap uint32) uint32 {
	return (1 << ((cap) & 31))
}

func capHasCap(caps []uint32, cap uint32, capPerm int) bool {
	return caps[capIndex(cap)+capPerm]&capMask(cap) == capMask(cap)
}

// perm = 0 -> permitted
// perm = 1 -> inheritable
func capSet(caps []uint32, cap uint32, capPerm int) ([]uint32, error) {
	if !capValid(cap) {
		return nil, fmt.Errorf("capability is invalid")
	}
	caps[capIndex(cap)+capPerm] = caps[capIndex(cap)+capPerm] | capMask(cap)
	return caps, nil
}

func capToText(cap []uint32) []string {
	out := []string{}
	for i := range CapabilityNames {
		capPermitted := capHasCap(cap, uint32(i), CAP_PERMITTED)
		capInheritable := capHasCap(cap, uint32(i), CAP_INHERITABLE)

		if capPermitted || capInheritable {
			var capStr strings.Builder
			capStr.WriteString(strings.ToLower(CapabilityNames[i]))
			capStr.WriteString("+")
			if capPermitted {
				capStr.WriteString("p")
			}
			if capInheritable {
				capStr.WriteString("i")
			}
			out = append(out, capStr.String())
		}
	}
	return out
}

func New(caps interface{}) ([]string, error) {
	cap := []string{}
	var capabilities []uint32
	var err error
	switch capsVal := caps.(type) {
	case []byte:
		capabilities, err = capsParse(capsVal, 20)
	case string:
		capabilities, err = capsParseFromText(capsVal)
	default:
		return cap, nil
	}

	if err != nil {
		return cap, nil
	}

	return capToText(capabilities), nil
}

func capsParse(caps []byte, capsLen uint32) ([]uint32, error) {
	if capsLen%4 != 0 {
		return nil, fmt.Errorf("capability length bad")
	}
	// capabilities are stored in uint32
	realCap := make([]uint32, capsLen/4)

	for i := 0; i < int(capsLen)/4; i++ {
		buf := bytes.NewBuffer(caps[i*4 : (i+1)*4])
		var num uint32
		err := binary.Read(buf, binary.LittleEndian, &num)
		if err != nil {
			return nil, err
		}
		realCap[i] = uint32(num)
	}
	// strip magic (first uint32 in the array)
	return realCap[1:], nil
}

// parse caps from string: 0x2000001,0x1000,0x0,0x0,0x0
// this is the format produced by e2tools and unsquashfs
func capsParseFromText(capsText string) ([]uint32, error) {
	capsInts := strings.Split(capsText, ",")
	capsParsedInts := make([]uint32, 5)
	for i, val := range capsInts {
		intVal, err := strconv.ParseUint(val[2:], 16, 32)
		if err != nil {
			return nil, err
		}
		capsParsedInts[i] = uint32(intVal)
	}
	capsBytes := make([]byte, 20)
	for i := range capsParsedInts {
		binary.LittleEndian.PutUint32(capsBytes[(i)*4:], capsParsedInts[i])
	}
	return capsParse(capsBytes, 20)
}

func CapsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aM := make(map[string]bool)
	for _, cap := range a {
		aM[cap] = true
	}
	bM := make(map[string]bool)
	for _, cap := range b {
		bM[cap] = true
	}

	for cap := range aM {
		if _, ok := bM[cap]; !ok {
			return false
		}
	}
	return true
}
