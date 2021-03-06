// Copyright (2013) Sandia Corporation.
// Under the terms of Contract DE-AC04-94AL85000 with Sandia Corporation,
// the U.S. Government retains certain rights in this software.

package main

import (
	"errors"
	"fmt"
	log "minilog"
	"ranges"
	"strconv"
	"strings"
	"time"
)

var cmdPower = &Command{
	UsageLine: "power [-r <reservation name>] [-n <nodes>] on/off/cycle",
	Short:     "power-cycle nodes or full reservation",
	Long: `
Power-cycle either a full reservation, or some nodes within a reservation owned by you.

Specify on or off to determine which power action should be taken.

Specify -r <reservation name> to indicate that the action should affect all nodes within the reservation.

Specify -n <nodes> to indicate that the action should affect the nodes listed. Only nodes in reservations belonging to you will be affected.
	`,
}

var powerR string
var powerN string

func init() {
	// break init cycle
	cmdPower.Run = runPower

	cmdPower.Flag.StringVar(&powerR, "r", "", "")
	cmdPower.Flag.StringVar(&powerN, "n", "", "")
}

func doPower(hosts []string, action string) error {
	log.Info("POWER	user=%v	nodes=%v	action=%v", User.Username, hosts, action)

	switch action {
	case "off":
		if igorConfig.PowerOffCommand == "" {
			return errors.New("power configuration missing")
		}

		return runAll(igorConfig.PowerOffCommand, hosts)
	case "cycle":
		if igorConfig.PowerOffCommand == "" {
			return errors.New("power configuration missing")
		}

		if err := runAll(igorConfig.PowerOffCommand, hosts); err != nil {
			return err
		}

		fallthrough
	case "on":
		if igorConfig.PowerOnCommand == "" {
			return errors.New("power configuration missing")
		}

		return runAll(igorConfig.PowerOnCommand, hosts)
	}

	return fmt.Errorf("invalid power operation: %v", action)
}

// Turn a node on or off
func runPower(cmd *Command, args []string) {
	if len(args) != 1 {
		log.Fatalln(cmdPower.UsageLine)
	}

	action := args[0]
	if action != "on" && action != "off" && action != "cycle" {
		log.Fatalln("must specify on, off, or cycle")
	}

	if powerR != "" {
		r := FindReservation(powerR)
		if r == nil {
			log.Fatal("reservation does not exist: %v", powerR)
		}

		if !r.IsActive(time.Now()) {
			log.Fatal("reservation is not active: %v", powerR)
		}

		if !r.IsWritable(User) {
			log.Fatal("insufficient privileges to power %v reservation: %v", action, powerR)
		}

		fmt.Printf("Powering %s reservation %s\n", action, powerR)
		doPower(r.Hosts, action)
	} else if powerN != "" {
		// The user specified some nodes. We need to verify they 'own' all those nodes.
		// Instead of looking through the reservations, we'll look at the current slice of the Schedule
		currentSched := Schedule[0]
		// Get the array of nodes the user specified
		rnge, _ := ranges.NewRange(igorConfig.Prefix, igorConfig.Start, igorConfig.End)
		nodes, _ := rnge.SplitRange(powerN)
		if len(nodes) == 0 {
			log.Fatal("Couldn't parse node specification %v\n", subW)
		}
		// make sure the range is valid
		if !checkValidNodeRange(nodes) {
			log.Fatalln("Invalid node range.")
		}

		// This will be the list of nodes to actually power on/off (in a user-owned reservation)
		var validatedNodes []string
		for _, n := range nodes {
			// Get rid of the prefix
			numstring := strings.TrimPrefix(n, igorConfig.Prefix)
			index, err := strconv.Atoi(numstring)
			if err != nil {
				log.Fatal("choked on a node named %v", n)
			}

			resID := currentSched.Nodes[index-1]
			for _, r := range Reservations {
				if r.ID == resID && r.IsWritable(User) {
					// Success! This node is in a reservation owned by the user
					validatedNodes = append(validatedNodes, n)
				}
			}
		}
		if len(validatedNodes) > 0 {
			unsplit, _ := rnge.UnsplitRange(validatedNodes)
			fmt.Printf("Powering %s nodes %s\n", action, unsplit)
			doPower(validatedNodes, action)
		} else {
			fmt.Printf("No nodes specified are in a reservation owned by the user\n")
		}
	}
}
