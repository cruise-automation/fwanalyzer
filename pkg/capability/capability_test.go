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
	"strings"
	"testing"
)

func TestCap(t *testing.T) {
	if !strings.EqualFold("cap_net_admin+p", capToText([]uint32{0x1000, 0x0, 0x0, 0x0})[0]) {
		t.Error("bad cap")
	}

	cap := []uint32{0, 0, 0, 0}
	cap, _ = capSet(cap, CAP_DAC_OVERRIDE, CAP_PERMITTED)
	cap, _ = capSet(cap, CAP_AUDIT_READ, CAP_INHERITABLE)
	if !capHasCap(cap, CAP_DAC_OVERRIDE, CAP_PERMITTED) {
		t.Error("bad cap")
	}
	if !capHasCap(cap, CAP_AUDIT_READ, CAP_INHERITABLE) {
		t.Error("bad cap")
	}
}

func TestCapsParse(t *testing.T) {
	caps, err := capsParse([]byte{0, 0, 0, 0, 0, 0x10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 20)
	if err != nil {
		t.Error(err)
	}
	if !capHasCap(caps, CAP_NET_ADMIN, CAP_PERMITTED) {
		t.Error("bad cap")
	}
}

func TestCapsStringParse(t *testing.T) {
	caps, err := capsParseFromText("0x2000001,0x1000,0x0,0x0,0x0")
	if err != nil {
		t.Error(err)
	}
	if !capHasCap(caps, CAP_NET_ADMIN, CAP_PERMITTED) {
		t.Error("bad cap")
	}
}

func TestCapMain(t *testing.T) {
	caps, err := New("0x2000001,0x1000,0x0,0x0,0x0")
	if err != nil {
		t.Error(err)
	}
	if !strings.EqualFold(caps[0], "cap_net_admin+p") {
		t.Error("bad cap")
	}

	caps2, err := New([]byte{0, 0, 0, 0, 0, 0x10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	if err != nil {
		t.Error(err)
	}
	if !strings.EqualFold(caps2[0], "cap_net_admin+p") {
		t.Error("bad cap")
	}
}
