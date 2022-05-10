/*
 * Copyright (C) 2021 The Zion Authors
 * This file is part of The Zion library.
 *
 * The Zion is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The Zion is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The Zion.  If not, see <http://www.gnu.org/licenses/>.
 */

package maas_config

import (
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/native"
	"github.com/ethereum/go-ethereum/contracts/native/contract"
	"github.com/ethereum/go-ethereum/contracts/native/utils"
	"github.com/ethereum/go-ethereum/log"
)

var (
	gasTable = map[string]uint64{
		MethodName:         0,
		MethodChangeOwner:  30000,
		MethodGetOwner:     0,
		MethodBlockAccount: 30000,
		MethodIsBlocked:    0,
		MethodGetBlacklist: 0,

		MethodEnableNodeWhite:    30000,
		MethodIsNodeWhiteEnabled: 0,
		MethodSetNodeWhite:       30000,
		MethodGetNodeWhitelist:   0,
		MethodIsInNodeWhite:      0,

		MethodEnableGasManage:    30000,
		MethodSetGasManager:      30000,
		MethodIsGasManageEnabled: 0,
		MethodIsGasManager:       0,
		MethodGetGasManagerList:  0,
	}
)

func InitMaasConfig() {
	InitABI()
	native.Contracts[this] = RegisterMaasConfigContract
}

func RegisterMaasConfigContract(s *native.NativeContract) {
	s.Prepare(ABI, gasTable)

	s.Register(MethodName, Name)
	s.Register(MethodChangeOwner, ChangeOwner)
	s.Register(MethodGetOwner, GetOwner)

	s.Register(MethodBlockAccount, BlockAccount)
	s.Register(MethodIsBlocked, IsBlocked)
	s.Register(MethodGetBlacklist, GetBlacklist)

	s.Register(MethodEnableNodeWhite, EnableNodeWhite)
	s.Register(MethodIsNodeWhiteEnabled, IsNodeWhiteEnabled)
	s.Register(MethodSetNodeWhite, SetNodeWhite)
	s.Register(MethodGetNodeWhitelist, GetNodeWhitelist)
	s.Register(MethodIsInNodeWhite, IsInNodeWhite)

	s.Register(MethodEnableGasManage, EnableGasManage)
	s.Register(MethodSetGasManager, SetGasManager)
	s.Register(MethodIsGasManageEnabled, IsGasManageEnabled)
	s.Register(MethodIsGasManager, IsGasManager)
	s.Register(MethodGetGasManagerList, GetGasManagerList)
}

func Name(s *native.NativeContract) ([]byte, error) {
	return new(MethodContractNameOutput).Encode()
}

// change owner
func ChangeOwner(s *native.NativeContract) ([]byte, error) {
	ctx := s.ContractRef().CurrentContext()
	caller := ctx.Caller

	// check authority
	if err := contract.ValidateOwner(s, caller); err != nil {
		return utils.ByteFailed, errors.New("invalid authority for caller")
	}

	currentOwner := getOwner(s)
	if currentOwner != common.EmptyAddress {
		if err := contract.ValidateOwner(s, currentOwner); err != nil {
			return utils.ByteFailed, errors.New("invalid authority for owner")
		}
	}

	// decode input
	input := new(MethodChangeOwnerInput)
	if err := input.Decode(ctx.Payload); err != nil {
		log.Trace("ChangeOwner", "decode input failed", err)
		return utils.ByteFailed, errors.New("invalid input")
	}

	// verify new owner address
	m := getAddressMap(s, blacklistKey)
	_, ok := m[input.Addr]
	if ok {
		err := errors.New("new owner address in blacklist")
		log.Trace("ChangeOwner", "invalid new owner", err)
		return utils.ByteFailed, err
	}

	// store owner
	set(s, ownerKey, input.Addr.Bytes())

	// emit event log
	if err := s.AddNotify(ABI, []string{EventChangeOwner}, common.BytesToHash(currentOwner.Bytes()), common.BytesToHash(input.Addr.Bytes())); err != nil {
		log.Trace("ChangeOwner", "emit event log failed", err)
		return utils.ByteFailed, errors.New("emit EventChangeOwner error")
	}

	return utils.ByteSuccess, nil
}

// get owner
func GetOwner(s *native.NativeContract) ([]byte, error) {
	output := &MethodAddressOutput{Addr: getOwner(s)}
	return output.Encode(MethodGetOwner)
}

func getOwner(s *native.NativeContract) common.Address {
	// get value
	value, _ := get(s, ownerKey)
	if len(value) == 0 {
		return common.EmptyAddress
	}
	return common.BytesToAddress(value)
}

func validateOwner(s *native.NativeContract) error {
	caller := s.ContractRef().CurrentContext().Caller
	if err := contract.ValidateOwner(s, caller); err != nil {
		log.Trace("validateOwner", "ValidateOwner caller failed", err)
		return errors.New("invalid authority for caller")
	}

	currentOwner := getOwner(s)
	if err := contract.ValidateOwner(s, currentOwner); err != nil {
		log.Trace("validateOwner", "ValidateOwner owner failed", err)
		return errors.New("invalid authority for owner")
	}
	return nil
}

// block account(add account to blacklist map) or unblock account
func BlockAccount(s *native.NativeContract) ([]byte, error) {
	ctx := s.ContractRef().CurrentContext()

	// check owner
	if err := validateOwner(s); err != nil {
		return utils.ByteFailed, err
	}

	// decode input
	input := new(MethodBlockAccountInput)
	if err := input.Decode(ctx.Payload); err != nil {
		log.Trace("blockAccount", "decode input failed", err)
		return utils.ByteFailed, errors.New("invalid input")
	}

	currentOwner := getOwner(s)
	if input.Addr == currentOwner {
		err := errors.New("block owner is forbidden")
		log.Trace("blockAccount", "block owner is forbidden", err)
		return utils.ByteFailed, err
	}

	m := getAddressMap(s, blacklistKey)
	if input.DoBlock {
		m[input.Addr] = struct{}{}
	} else {
		delete(m, input.Addr)
	}

	value, err := json.Marshal(m)
	if err != nil {
		log.Trace("blockAccount", "encode value failed", err)
		return utils.ByteFailed, errors.New("encode value failed")
	}
	set(s, blacklistKey, value)

	// emit event log
	if err := s.AddNotify(ABI, []string{EventBlockAccount}, common.BytesToHash(input.Addr.Bytes()), input.DoBlock); err != nil {
		log.Trace("blockAccount", "emit event log failed", err)
		return utils.ByteFailed, errors.New("emit EventBlockAccount error")
	}

	return utils.ByteSuccess, nil
}

func getAddressMap(s *native.NativeContract, key []byte) map[common.Address]struct{} {
	value, _ := get(s, key)
	m := make(map[common.Address]struct{})
	if len(value) > 0 {
		if err := json.Unmarshal(value, &m); err != nil {
			log.Trace("getAddressMap", "decode value failed", err)
		}
	}
	return m
}

// check if account is blocked
func IsBlocked(s *native.NativeContract) ([]byte, error) {
	ctx := s.ContractRef().CurrentContext()

	// decode input
	input := new(MethodIsBlockedInput)
	if err := input.Decode(ctx.Payload); err != nil {
		log.Trace("IsBlocked", "decode input failed", err)
		return utils.ByteFailed, errors.New("invalid input")
	}

	// get value
	m := getAddressMap(s, blacklistKey)
	_, ok := m[input.Addr]
	output := &MethodBoolOutput{Success: ok}

	return output.Encode(MethodIsBlocked)
}

// get blacklist json
func GetBlacklist(s *native.NativeContract) ([]byte, error) {
	// get value
	m := getAddressMap(s, blacklistKey)
	list := make([]common.Address, 0, len(m))
	for key := range m {
		list = append(list, key)
	}
	result, _ := json.Marshal(list)
	output := &MethodStringOutput{Result: string(result)}
	return output.Encode(MethodGetBlacklist)
}

// enable node whitelist
func EnableNodeWhite(s *native.NativeContract) ([]byte, error) {
	ctx := s.ContractRef().CurrentContext()

	// check owner
	if err := validateOwner(s); err != nil {
		return utils.ByteFailed, err
	}

	// decode input
	input := new(MethodEnableNodeWhiteInput)
	if err := input.Decode(ctx.Payload); err != nil {
		log.Trace("EnableNodeWhite", "decode input failed", err)
		return utils.ByteFailed, errors.New("invalid input")
	}

	// set enable status
	if input.DoEnable {
		set(s, nodeWhiteEnableKey, utils.BYTE_TRUE)
	} else {
		del(s, nodeWhiteEnableKey)
	}

	// emit event log
	if err := s.AddNotify(ABI, []string{EventEnableNodeWhite}, input.DoEnable); err != nil {
		log.Trace("EnableNodeWhite", "emit event log failed", err)
		return utils.ByteFailed, errors.New("emit EventEnableNodeWhite error")
	}

	return utils.ByteSuccess, nil
}

// check if node whitelist is enabled
func IsNodeWhiteEnabled(s *native.NativeContract) ([]byte, error) {
	// get value
	value, _ := get(s, nodeWhiteEnableKey)
	output := &MethodBoolOutput{Success: len(value) > 0}
	return output.Encode(MethodIsNodeWhiteEnabled)
}

// set node whitelist for p2p connection
func SetNodeWhite(s *native.NativeContract) ([]byte, error) {
	ctx := s.ContractRef().CurrentContext()

	// check owner
	if err := validateOwner(s); err != nil {
		return utils.ByteFailed, err
	}

	// decode input
	input := new(MethodSetNodeWhiteInput)
	if err := input.Decode(ctx.Payload); err != nil {
		log.Trace("setNodeWhite", "decode input failed", err)
		return utils.ByteFailed, errors.New("invalid input")
	}

	m := getAddressMap(s, nodeWhitelistKey)
	if input.IsWhite {
		m[input.Addr] = struct{}{}
	} else {
		delete(m, input.Addr)
	}

	value, err := json.Marshal(m)
	if err != nil {
		log.Trace("setNodeWhite", "encode value failed", err)
		return utils.ByteFailed, errors.New("encode value failed")
	}
	set(s, nodeWhitelistKey, value)

	// emit event log
	if err := s.AddNotify(ABI, []string{EventSetNodeWhite}, common.BytesToHash(input.Addr.Bytes()), input.IsWhite); err != nil {
		log.Trace("setNodeWhite", "emit event log failed", err)
		return utils.ByteFailed, errors.New("emit EventSetNodeWhite error")
	}

	return utils.ByteSuccess, nil
}

// check if address is in node whitelist
func IsInNodeWhite(s *native.NativeContract) ([]byte, error) {
	ctx := s.ContractRef().CurrentContext()

	// decode input
	input := new(MethodIsInNodeWhiteInput)
	if err := input.Decode(ctx.Payload); err != nil {
		log.Trace("IsInNodeWhite", "decode input failed", err)
		return utils.ByteFailed, errors.New("invalid input")
	}

	// get value
	m := getAddressMap(s, nodeWhitelistKey)
	_, ok := m[input.Addr]
	output := &MethodBoolOutput{Success: ok}

	return output.Encode(MethodIsInNodeWhite)
}

// get node whitelist json
func GetNodeWhitelist(s *native.NativeContract) ([]byte, error) {
	// get value
	m := getAddressMap(s, nodeWhitelistKey)
	list := make([]common.Address, 0, len(m))
	for key := range m {
		list = append(list, key)
	}
	result, _ := json.Marshal(list)
	output := &MethodStringOutput{Result: string(result)}
	return output.Encode(MethodGetNodeWhitelist)
}

// enable gas manage
func EnableGasManage(s *native.NativeContract) ([]byte, error) {
	ctx := s.ContractRef().CurrentContext()

	// check owner
	if err := validateOwner(s); err != nil {
		return utils.ByteFailed, err
	}

	// decode input
	input := new(MethodEnableGasManageInput)
	if err := input.Decode(ctx.Payload); err != nil {
		log.Trace("EnableGasManage", "decode input failed", err)
		return utils.ByteFailed, errors.New("invalid input")
	}

	// set enable status
	if input.DoEnable {
		set(s, gasManageEnableKey, utils.BYTE_TRUE)
	} else {
		del(s, gasManageEnableKey)
	}

	// emit event log
	if err := s.AddNotify(ABI, []string{EventEnableGasManage}, input.DoEnable); err != nil {
		log.Trace("EnableGasManage", "emit event log failed", err)
		return utils.ByteFailed, errors.New("emit EventEnableGasManage error")
	}

	return utils.ByteSuccess, nil
}

// check if gas manage is enabled
func IsGasManageEnabled(s *native.NativeContract) ([]byte, error) {
	// get value
	value, _ := get(s, gasManageEnableKey)
	output := &MethodBoolOutput{Success: len(value) > 0}
	return output.Encode(MethodIsGasManageEnabled)
}

// set gas manager address
func SetGasManager(s *native.NativeContract) ([]byte, error) {
	ctx := s.ContractRef().CurrentContext()

	// check owner
	if err := validateOwner(s); err != nil {
		return utils.ByteFailed, err
	}

	// decode input
	input := new(MethodSetGasManagerInput)
	if err := input.Decode(ctx.Payload); err != nil {
		log.Trace("SetGasManager", "decode input failed", err)
		return utils.ByteFailed, errors.New("invalid input")
	}

	m := getAddressMap(s, gasManagerListKey)
	if input.IsManager {
		m[input.Addr] = struct{}{}
	} else {
		delete(m, input.Addr)
	}

	value, err := json.Marshal(m)
	if err != nil {
		log.Trace("SetGasManager", "encode value failed", err)
		return utils.ByteFailed, errors.New("encode value failed")
	}
	set(s, gasManagerListKey, value)

	// emit event log
	if err := s.AddNotify(ABI, []string{EventSetGasManager}, common.BytesToHash(input.Addr.Bytes()), input.IsManager); err != nil {
		log.Trace("SetGasManager", "emit event log failed", err)
		return utils.ByteFailed, errors.New("emit EventSetGasManager error")
	}

	return utils.ByteSuccess, nil
}

// check if address is in gas manager list
func IsGasManager(s *native.NativeContract) ([]byte, error) {
	ctx := s.ContractRef().CurrentContext()

	// decode input
	input := new(MethodIsGasManagerInput)
	if err := input.Decode(ctx.Payload); err != nil {
		log.Trace("IsGasManager", "decode input failed", err)
		return utils.ByteFailed, errors.New("invalid input")
	}

	// get value
	m := getAddressMap(s, gasManagerListKey)
	_, ok := m[input.Addr]
	output := &MethodBoolOutput{Success: ok}

	return output.Encode(MethodIsGasManager)
}

// get gas manager list json
func GetGasManagerList(s *native.NativeContract) ([]byte, error) {
	// get value
	m := getAddressMap(s, gasManagerListKey)
	list := make([]common.Address, 0, len(m))
	for key := range m {
		list = append(list, key)
	}
	result, _ := json.Marshal(list)
	output := &MethodStringOutput{Result: string(result)}
	return output.Encode(MethodGetGasManagerList)
}
