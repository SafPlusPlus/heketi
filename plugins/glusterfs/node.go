//
// Copyright (c) 2014 The heketi Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package glusterfs

import (
	"errors"
	"fmt"
	"github.com/lpabon/godbc"
	"github.com/lpabon/heketi/requests"
	"github.com/lpabon/heketi/utils"
	"github.com/lpabon/heketi/utils/ssh"
	"strconv"
	"strings"
)

const (
	VGDISPLAY_SIZE_KB                  = 11
	VGDISPLAY_PHYSICAL_EXTENT_SIZE     = 12
	VGDISPLAY_TOTAL_NUMBER_EXTENTS     = 13
	VGDISPLAY_ALLOCATED_NUMBER_EXTENTS = 14
	VGDISPLAY_FREE_NUMBER_EXTENTS      = 15
)

type NodeDB struct {
	Info requests.NodeInfoResp
	requests.Storage
}

func NewNodeDB(v *requests.NodeAddRequest) *NodeDB {

	node := &NodeDB{}
	node.Info.Id = utils.GenUUID()
	node.Info.Name = v.Name
	node.Info.Zone = v.Zone
	node.Info.VolumeGroups = make([]requests.LvmVolumeGroup, 0)
	node.Disks = v.Disks
	node.VolumeGroups = v.VolumeGroups

	return node
}

func (n *NodeDB) GetVgSizeFromNode() error {

	// Just for now, it will work wih https://github.com/lpabon/vagrant-gfsm
	sshexec := ssh.NewSshExecWithKeyFile("vagrant", "insecure_private_key")
	godbc.Check(sshexec != nil)

	commands := []string{
		fmt.Sprintf("sudo vgdisplay -c %v", n.VolumeGroups[0]),
	}

	b, err := sshexec.ConnectAndExec(n.Info.Name+":22", commands, nil)
	if err != nil {
		return err
	}

	vginfo := strings.Split(b[0], ":")

	// See vgdisplay manpage
	if len(vginfo) < 17 {
		return errors.New("vgdisplay returned an invalid string")
	}

	n.Info.Storage.Total, err =
		strconv.ParseUint(vginfo[VGDISPLAY_SIZE_KB], 10, 64)
	if err != nil {
		return err
	}

	extent_size, err :=
		strconv.ParseUint(vginfo[VGDISPLAY_PHYSICAL_EXTENT_SIZE], 10, 64)
	if err != nil {
		return err
	}

	free_extents, err :=
		strconv.ParseUint(vginfo[VGDISPLAY_FREE_NUMBER_EXTENTS], 10, 64)
	if err != nil {
		return err
	}

	alloc_extents, err :=
		strconv.ParseUint(vginfo[VGDISPLAY_ALLOCATED_NUMBER_EXTENTS], 10, 64)
	if err != nil {
		return err
	}

	n.Info.Storage.Free = free_extents * extent_size
	n.Info.Storage.Used = alloc_extents * extent_size

	return nil
}