//  Copyright (c) 2019 Cisco and/or its affiliates.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at:
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package vpp2001_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"go.ligato.io/cn-infra/v2/logging/logrus"

	"go.ligato.io/vpp-agent/v3/plugins/vpp/binapi/vpp1904/ip"
	vpp_arp "go.ligato.io/vpp-agent/v3/plugins/vpp/binapi/vpp2001/arp"
	"go.ligato.io/vpp-agent/v3/plugins/vpp/ifplugin/ifaceidx"
	"go.ligato.io/vpp-agent/v3/plugins/vpp/l3plugin/vppcalls"
	"go.ligato.io/vpp-agent/v3/plugins/vpp/l3plugin/vppcalls/vpp2001"
	"go.ligato.io/vpp-agent/v3/plugins/vpp/vppmock"
)

// Test enable/disable proxy arp
func TestProxyArp(t *testing.T) {
	ctx, ifIndexes, pArpHandler := pArpTestSetup(t)
	defer ctx.TeardownTestCtx()

	ifIndexes.Put("if1", &ifaceidx.IfaceMetadata{SwIfIndex: 1})

	ctx.MockVpp.MockReply(&vpp_arp.ProxyArpIntfcEnableDisableReply{})
	err := pArpHandler.EnableProxyArpInterface("if1")
	Expect(err).To(Succeed())

	ctx.MockVpp.MockReply(&vpp_arp.ProxyArpIntfcEnableDisableReply{})
	err = pArpHandler.DisableProxyArpInterface("if1")
	Expect(err).To(Succeed())

	ctx.MockVpp.MockReply(&vpp_arp.ProxyArpIntfcEnableDisableReply{Retval: 1})
	err = pArpHandler.DisableProxyArpInterface("if1")
	Expect(err).NotTo(BeNil())
}

func TestProxyArpRange(t *testing.T) {
	ctx, _, pArpHandler := pArpTestSetup(t)
	defer ctx.TeardownTestCtx()

	t.Run("Add: success case", func(t *testing.T) {
		ctx.MockVpp.MockReply(&ip.ProxyArpAddDelReply{Retval: 0})

		Expect(pArpHandler.AddProxyArpRange(
			[]byte{192, 168, 10, 20}, []byte{192, 168, 10, 30}, 0,
		)).To(Succeed())
	})

	testAddProxyARPRangeError := func(firstIP, lastIP []byte, vrf uint32) {
		t.Helper()

		ctx.MockVpp.MockReply(&ip.ProxyArpAddDelReply{Retval: 0})
		Expect(pArpHandler.AddProxyArpRange(firstIP, lastIP, vrf)).ToNot(Succeed())

		//Get mocked reply, since VPP call should not happen before.
		Expect(
			ctx.MockVPPClient.SendRequest(&ip.ProxyArpAddDel{}).ReceiveReply(&ip.ProxyArpAddDelReply{}),
		).To(Succeed())
	}
	t.Run("Add: error cases", func(t *testing.T) {
		// Bad first IP address.
		testAddProxyARPRangeError([]byte{192, 168, 20}, []byte{192, 168, 10, 30}, 0)
		// Bad last IP address.
		testAddProxyARPRangeError([]byte{192, 168, 10, 20}, []byte{192, 168, 30}, 0)
		// Bad both IP addresses.
		testAddProxyARPRangeError([]byte{192, 168, 20}, []byte{192, 168, 30}, 0)
	})

	t.Run("Delete: success case", func(t *testing.T) {
		ctx.MockVpp.MockReply(&ip.ProxyArpAddDelReply{Retval: 0})

		Expect(pArpHandler.DeleteProxyArpRange(
			[]byte{192, 168, 10, 20}, []byte{192, 168, 10, 30}, 0,
		)).To(Succeed())
	})

	testDelProxyARPRangeError := func(firstIP, lastIP []byte, vrf uint32) {
		t.Helper()

		ctx.MockVpp.MockReply(&ip.ProxyArpAddDelReply{Retval: 0})
		Expect(pArpHandler.DeleteProxyArpRange(firstIP, lastIP, vrf)).ToNot(Succeed())

		//Get mocked reply, since VPP call should not happen before.
		Expect(
			ctx.MockVPPClient.SendRequest(&ip.ProxyArpAddDel{}).ReceiveReply(&ip.ProxyArpAddDelReply{}),
		).To(Succeed())
	}
	t.Run("Delete: error cases", func(t *testing.T) {
		// Bad first IP address.
		testDelProxyARPRangeError([]byte{192, 168, 20}, []byte{192, 168, 10, 30}, 0)
		// Bad last IP address.
		testDelProxyARPRangeError([]byte{192, 168, 10, 20}, []byte{192, 168, 30}, 0)
		// Bad both IP addresses.
		testDelProxyARPRangeError([]byte{192, 168, 20}, []byte{192, 168, 30}, 0)
	})

	// Test retval in "add" scenario.
	ctx.MockVpp.MockReply(&ip.ProxyArpAddDelReply{Retval: 1})
	Expect(pArpHandler.AddProxyArpRange(
		[]byte{192, 168, 10, 20}, []byte{192, 168, 10, 30}, 0,
	)).ToNot(Succeed())

	// Test retval in "delete" scenario.
	ctx.MockVpp.MockReply(&ip.ProxyArpAddDelReply{Retval: 1})
	Expect(pArpHandler.DeleteProxyArpRange(
		[]byte{192, 168, 10, 20}, []byte{192, 168, 10, 30}, 0,
	)).ToNot(Succeed())
}

func pArpTestSetup(t *testing.T) (*vppmock.TestCtx, ifaceidx.IfaceMetadataIndexRW, vppcalls.ProxyArpVppAPI) {
	ctx := vppmock.SetupTestCtx(t)
	log := logrus.NewLogger("test-log")
	ifIndexes := ifaceidx.NewIfaceIndex(logrus.NewLogger("test"), "test")
	pArpHandler := vpp2001.NewProxyArpVppHandler(ctx.MockChannel, ifIndexes, log)
	return ctx, ifIndexes, pArpHandler
}
