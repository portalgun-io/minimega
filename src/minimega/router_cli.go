// Copyright (2016) Sandia Corporation.
// Under the terms of Contract DE-AC04-94AL85000 with Sandia Corporation,
// the U.S. Government retains certain rights in this software.

package main

import (
	"fmt"
	"minicli"
	"strconv"
)

var routerCLIHandlers = []minicli.Handler{
	{ // router
		HelpShort: "",
		HelpLong:  ``,
		Patterns: []string{
			"router <vm>",
			"router <vm> <commit,>",
			"router <vm> <log,> <level,> <fatal,error,warn,info,debug>",
			"router <vm> <interface,> <network> <IPv4/MASK or IPv6/MASK or dhcp>",
			"router <vm> <dhcp,> <listen address> <range,> <low address> <high address>",
			"router <vm> <dhcp,> <listen address> <router,> <router address>",
			"router <vm> <dhcp,> <listen address> <dns,> <address>",
			"router <vm> <dhcp,> <listen address> <static,> <mac> <ip>",
			"router <vm> <dns,> <ip> <hostname>",
			"router <vm> <ra,> <subnet>",
			"router <vm> <route,> <static,> <network> <next-hop>",
			"router <vm> <route,> <ospf,> <area> <interface>",
		},
		Call: wrapBroadcastCLI(cliRouter),
	},
	{ // clear router
		HelpShort: "",
		HelpLong:  ``,
		Patterns: []string{
			"clear router <vm>",
			"clear router <vm> <interface,>",
			"clear router <vm> <interface,> <network>",
			"clear router <vm> <interface,> <network> <IPv4/MASK or IPv6/MASK or dhcp>",
			"clear router <vm> <dhcp,>",
			"clear router <vm> <dhcp,> <listen address>",
			"clear router <vm> <dhcp,> <listen address> <range,>",
			"clear router <vm> <dhcp,> <listen address> <router,>",
			"clear router <vm> <dhcp,> <listen address> <dns,>",
			"clear router <vm> <dhcp,> <listen address> <static,>",
			"clear router <vm> <dhcp,> <listen address> <static,> <mac>",
			"clear router <vm> <dns,>",
			"clear router <vm> <dns,> <ip>",
			"clear router <vm> <ra,>",
			"clear router <vm> <ra,> <subnet>",
			"clear router <vm> <route,>",
			"clear router <vm> <route,> <static,>",
			"clear router <vm> <route,> <static,> <network>",
			"clear router <vm> <route,> <ospf,>",
			"clear router <vm> <route,> <ospf,> <area>",
			"clear router <vm> <route,> <ospf,> <area> <interface>",
		},
		Call: wrapBroadcastCLI(cliClearRouter),
	},
}

func cliRouter(c *minicli.Command, resp *minicli.Response) error {
	vmName := c.StringArgs["vm"]

	vm := vms.FindVM(vmName)
	if vm == nil {
		return fmt.Errorf("no such vm %v", vmName)
	}

	if vmName != "" && len(c.BoolArgs) == 0 { // a summary of a specific router
		rtr := FindRouter(vm)
		if rtr == nil {
			return fmt.Errorf("vm %v not a router", vmName)
		}
		resp.Response = rtr.String()
	}

	rtr := FindOrCreateRouter(vm)

	if c.BoolArgs["commit"] {
		return rtr.Commit()
	} else if c.BoolArgs["log"] {
		var level string
		if c.BoolArgs["fatal"] {
			level = "fatal"
		} else if c.BoolArgs["error"] {
			level = "error"
		} else if c.BoolArgs["warn"] {
			level = "warn"
		} else if c.BoolArgs["info"] {
			level = "info"
		} else if c.BoolArgs["debug"] {
			level = "debug"
		}
		rtr.LogLevel(level)
		return nil
	} else if c.BoolArgs["interface"] {
		network, err := strconv.Atoi(c.StringArgs["network"])
		if err != nil {
			return fmt.Errorf("invalid network: %v : %v", c.StringArgs["network"], err)
		}
		ip := c.StringArgs["IPv4/MASK"]

		return rtr.InterfaceAdd(network, ip)
	} else if c.BoolArgs["dhcp"] {
		addr := c.StringArgs["listen"]

		if c.BoolArgs["range"] {
			low := c.StringArgs["low"]
			high := c.StringArgs["high"]
			return rtr.DHCPAddRange(addr, low, high)
		} else if c.BoolArgs["router"] {
			r := c.StringArgs["router"]
			return rtr.DHCPAddRouter(addr, r)
		} else if c.BoolArgs["dns"] {
			dns := c.StringArgs["address"]
			return rtr.DHCPAddDNS(addr, dns)
		} else if c.BoolArgs["static"] {
			mac := c.StringArgs["mac"]
			ip := c.StringArgs["ip"]
			return rtr.DHCPAddStatic(addr, mac, ip)
		}
	} else if c.BoolArgs["dns"] {
		ip := c.StringArgs["ip"]
		hostname := c.StringArgs["hostname"]
		rtr.DNSAdd(ip, hostname)
		return nil
	} else if c.BoolArgs["ra"] {
		subnet := c.StringArgs["subnet"]
		rtr.RADAdd(subnet)
		return nil
	} else if c.BoolArgs["route"] {
		if c.BoolArgs["static"] {
			network := c.StringArgs["network"]
			nh := c.StringArgs["next-hop"]
			rtr.RouteStaticAdd(network, nh)
			return nil
		} else if c.BoolArgs["ospf"] {
			area := c.StringArgs["area"]
			iface := c.StringArgs["interface"]
			rtr.RouteOSPFAdd(area, iface)
		}
	}

	return nil
}

func cliClearRouter(c *minicli.Command, resp *minicli.Response) error {
	vmName := c.StringArgs["vm"]

	vm := vms.FindVM(vmName)
	if vm == nil {
		return fmt.Errorf("no such vm %v", vmName)
	}
	rtr := FindRouter(vm)
	if rtr == nil {
		return fmt.Errorf("no such router %v", vmName)
	}

	if c.BoolArgs["interface"] {
		network := c.StringArgs["network"]
		ip := c.StringArgs["IPv4/MASK"]

		err := rtr.InterfaceDel(network, ip)
		if err != nil {
			return err
		}
	} else if c.BoolArgs["dhcp"] {
		addr := c.StringArgs["listen"]

		if addr == "" {
			// clear all of it
			rtr.dhcp = make(map[string]*dhcp)
			return nil
		}

		d, ok := rtr.dhcp[addr]
		if !ok {
			return fmt.Errorf("no such address: %v", addr)
		}

		if c.BoolArgs["range"] {
			d.low = ""
			d.high = ""
		} else if c.BoolArgs["router"] {
			d.router = ""
		} else if c.BoolArgs["dns"] {
			d.dns = ""
		} else if c.BoolArgs["static"] {
			mac := c.StringArgs["mac"]
			if mac == "" {
				d.static = make(map[string]string)
			} else {
				if _, ok := d.static[mac]; ok {
					delete(d.static, mac)
				} else {
					return fmt.Errorf("no such mac: %v", mac)
				}
			}
		} else {
			delete(rtr.dhcp, addr)
		}
	} else if c.BoolArgs["dns"] {
		ip := c.StringArgs["ip"]
		return rtr.DNSDel(ip)
	} else if c.BoolArgs["ra"] {
		subnet := c.StringArgs["subnet"]
		return rtr.RADDel(subnet)
	} else if c.BoolArgs["route"] {
		if c.BoolArgs["static"] {
			network := c.StringArgs["network"]
			return rtr.RouteStaticDel(network)
		} else if c.BoolArgs["ospf"] {
			area := c.StringArgs["area"]
			iface := c.StringArgs["interface"]
			return rtr.RouteOSPFDel(area, iface)
		} else {
			// clear all routes on all protocols
			rtr.RouteStaticDel("")
			rtr.RouteOSPFDel("", "")
		}
	} else {
		// remove everything about this router
		err := rtr.InterfaceDel("", "")
		if err != nil {
			return err
		}
		rtr.DNSDel("")
		rtr.RADDel("")
		rtr.RouteStaticDel("")
		rtr.RouteOSPFDel("", "")
		rtr.dhcp = make(map[string]*dhcp)
	}
	return nil
}
