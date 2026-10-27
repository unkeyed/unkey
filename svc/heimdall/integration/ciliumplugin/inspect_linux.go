package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

func inspect(directory string) (err error) {
	entry, err := link.LoadPinnedLink(filepath.Join(directory, "links", targetName), nil)
	if err != nil {
		return fmt.Errorf("open entrypoint link: %w", err)
	}
	defer func() { err = errors.Join(err, entry.Close()) }()
	info, err := entry.Info()
	if err != nil {
		return err
	}
	if info.TCX() == nil || ebpf.AttachType(info.TCX().AttachType) != ebpf.AttachTCXIngress {
		return errors.New("entrypoint is not an ingress TCX attachment")
	}
	query := link.QueryOptions{Target: int(info.TCX().Ifindex), Attach: ebpf.AttachTCXIngress, QueryFlags: 0}
	attached, err := link.QueryPrograms(query)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(attached.Programs, func(program link.AttachedProgram) bool { return program.ID == info.Program }) {
		return errors.New("pinned entrypoint is not attached")
	}
	root := filepath.Join(directory, "plugin_pins", "heimdall")
	observations, err := ebpf.LoadPinnedMap(filepath.Join(root, "pins", "observations"), nil)
	if err != nil {
		return fmt.Errorf("observations unavailable, not zero: %w", err)
	}
	defer func() { err = errors.Join(err, observations.Close()) }()
	mapInfo, err := observations.Info()
	if err != nil {
		return err
	}
	mapID, ok := mapInfo.ID()
	if !ok {
		return errors.New("observation map ID unavailable")
	}
	for _, name := range []string{"__pre_hook_heimdall__", "__post_hook_heimdall__"} {
		if err := verifyHook(filepath.Join(root, "hooks", targetName+"_"+name), info.Program, mapID); err != nil {
			return err
		}
	}
	var values bpfObservation
	if err := observations.Lookup(uint32(0), &values); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		Attachment string
		Identity   string
		TargetID   ebpf.ProgramID
		MapID      ebpf.MapID
		Counters   bpfObservation
	}{Attachment: "attached-at-query", Identity: "unverified", TargetID: info.Program, MapID: mapID, Counters: values})
}

func verifyHook(path string, target ebpf.ProgramID, mapID ebpf.MapID) (err error) {
	hook, err := link.LoadPinnedLink(path, nil)
	if err != nil {
		return fmt.Errorf("open hook: %w", err)
	}
	defer func() { err = errors.Join(err, hook.Close()) }()
	info, err := hook.Info()
	if err != nil {
		return err
	}
	if info.Tracing() == nil || info.Tracing().TargetObjId != uint32(target) {
		return errors.New("hook does not target the attached entrypoint")
	}
	program, err := ebpf.NewProgramFromID(info.Program)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, program.Close()) }()
	programInfo, err := program.Info()
	if err != nil {
		return err
	}
	maps, ok := programInfo.MapIDs()
	if !ok || !slices.Contains(maps, mapID) {
		return errors.New("hook does not reference the observation map")
	}
	return nil
}
