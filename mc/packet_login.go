package mc

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"
)

type HandshakeState byte

const (
	UNKNOWN_STATE HandshakeState = iota
	STATUS
	LOGIN
)

func RequestState(n int) HandshakeState {
	var t HandshakeState
	switch n {
	case 1:
		t = STATUS
	case 2:
		t = LOGIN
	default:
		t = UNKNOWN_STATE
	}
	return t
}

func (t HandshakeState) String() string {
	var text string
	switch t {
	case UNKNOWN_STATE:
		text = "Unknown"
	case STATUS:
		text = "Status"
	case LOGIN:
		text = "Login"
	}
	return text
}

type McTypesHandshake struct {
	ProtocolVersion VarInt
	ServerAddress   String
	ServerPort      UnsignedShort
	NextState       VarInt
}

type ServerBoundHandshake struct {
	ProtocolVersion int
	ServerAddress   string
	ServerPort      int16
	NextState       int
}

func (pk ServerBoundHandshake) Marshal() Packet {
	return MarshalPacket(
		ServerBoundHandshakePacketID,
		VarInt(pk.ProtocolVersion),
		String(pk.ServerAddress),
		UnsignedShort(pk.ServerPort),
		VarInt(pk.NextState),
	)
}

func UnmarshalServerBoundHandshake(packet Packet) (ServerBoundHandshake, error) {
	var pk McTypesHandshake
	var hs ServerBoundHandshake

	if packet.ID != ServerBoundHandshakePacketID {
		return hs, ErrInvalidPacketID
	}

	if err := packet.Scan(
		&pk.ProtocolVersion,
		&pk.ServerAddress,
		&pk.ServerPort,
		&pk.NextState,
	); err != nil {
		return hs, err
	}

	hs = ServerBoundHandshake{
		ProtocolVersion: int(pk.ProtocolVersion),
		ServerAddress:   string(pk.ServerAddress),
		ServerPort:      int16(pk.ServerPort),
		NextState:       int(pk.NextState),
	}
	return hs, nil
}

func UnmarshalServerBoundHandshake2(packet Packet) (ServerBoundHandshake, error) {
	var hs ServerBoundHandshake

	if packet.ID != ServerBoundHandshakePacketID {
		return hs, ErrInvalidPacketID
	}

	buf := bytes.NewBuffer(packet.Data)
	var err error
	hs.ProtocolVersion, err = ReadVarInt_ByteReader(buf)
	if err != nil {
		return hs, err
	}
	hs.ServerAddress, err = ReadString_ByteReader(buf)
	if err != nil {
		return hs, err
	}
	hs.ServerPort, err = ReadShot_ByteReader(buf)
	if err != nil {
		return hs, err
	}
	hs.NextState, err = ReadVarInt_ByteReader(buf)
	if err != nil {
		return hs, err
	}
	return hs, nil
}

func UnmarshalServerBoundHandshake_ByteReader(r io.ByteReader) (ServerBoundHandshake, error) {
	var hs ServerBoundHandshake
	packetID, err := r.ReadByte()
	if err != nil {
		return hs, err
	}
	if packetID != ServerBoundHandshakePacketID {
		return hs, ErrInvalidPacketID
	}

	hs.ProtocolVersion, err = ReadVarInt_ByteReader(r)
	if err != nil {
		return hs, err
	}
	hs.ServerAddress, err = ReadString_ByteReader(r)
	if err != nil {
		return hs, err
	}
	hs.ServerPort, err = ReadShot_ByteReader(r)
	if err != nil {
		return hs, err
	}
	hs.NextState, err = ReadVarInt_ByteReader(r)
	if err != nil {
		return hs, err
	}
	return hs, nil
}

func (hs ServerBoundHandshake) State() HandshakeState {
	var state HandshakeState
	switch hs.NextState {
	case 1:
		state = STATUS
	case 2:
		state = LOGIN
	default:
		state = UNKNOWN_STATE
	}
	return state
}

func (hs ServerBoundHandshake) IsStatusRequest() bool {
	return VarInt(hs.NextState) == HandshakeStatusState
}

func (hs ServerBoundHandshake) IsLoginRequest() bool {
	return VarInt(hs.NextState) == HandshakeLoginState
}

func (hs ServerBoundHandshake) IsForgeAddress() bool {
	addr := string(hs.ServerAddress)
	return len(strings.Split(addr, ForgeSeparator)) > 1
}

func (hs ServerBoundHandshake) IsRealIPAddress() bool {
	addr := string(hs.ServerAddress)
	return len(strings.Split(addr, RealIPSeparator)) > 1
}

func (hs ServerBoundHandshake) ParseServerAddress() string {
	addr := hs.ServerAddress
	addr = strings.Split(addr, ForgeSeparator)[0]
	addr = strings.Split(addr, RealIPSeparator)[0]
	return addr
}

func (hs *ServerBoundHandshake) UpgradeToOldRealIP(clientAddr string) {
	hs.UpgradeToOldRealIP_WithTime(clientAddr, time.Now())
}

func (hs *ServerBoundHandshake) UpgradeToOldRealIP_WithTime(clientAddr string, stamp time.Time) {
	if hs.IsRealIPAddress() {
		return
	}

	addr := string(hs.ServerAddress)
	addrWithForge := strings.SplitN(addr, ForgeSeparator, 3)

	addr = fmt.Sprintf("%s///%s///%d", addrWithForge[0], clientAddr, stamp.Unix())

	if len(addrWithForge) > 1 {
		addr = fmt.Sprintf("%s\x00%s\x00", addr, addrWithForge[1])
	}

	hs.ServerAddress = addr
}

func (hs *ServerBoundHandshake) UpgradeToNewRealIP(clientAddr string, key *ecdsa.PrivateKey) error {
	hs.UpgradeToOldRealIP(clientAddr)
	text := hs.ServerAddress
	hash := sha512.Sum512([]byte(text))
	bytes, err := ecdsa.SignASN1(rand.Reader, key, hash[:])
	if err != nil {
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(bytes)
	addr := fmt.Sprintf("%s///%s", hs.ServerAddress, encoded)
	hs.ServerAddress = addr
	return nil
}

const ServerBoundLoginStartPacketID byte = 0x00

type ServerLoginStart struct {
	Name String
}

func (pk ServerLoginStart) Marshal() Packet {
	return MarshalPacket(ServerBoundLoginStartPacketID, pk.Name)
}

func UnmarshalServerBoundLoginStart(packet Packet) (ServerLoginStart, error) {
	var pk ServerLoginStart

	if packet.ID != ServerBoundLoginStartPacketID {
		return pk, ErrInvalidPacketID
	}

	if err := packet.Scan(&pk.Name); err != nil {
		return pk, err
	}

	return pk, nil
}

const ClientBoundDisconnectPacketID byte = 0x00

type ClientBoundDisconnect struct {
	Reason Chat
}

func (pk ClientBoundDisconnect) Marshal() Packet {
	return MarshalPacket(
		ClientBoundDisconnectPacketID,
		pk.Reason,
	)
}

func UnmarshalClientDisconnect(packet Packet) (ClientBoundDisconnect, error) {
	var pk ClientBoundDisconnect

	if packet.ID != ClientBoundDisconnectPacketID {
		return pk, ErrInvalidPacketID
	}

	err := packet.Scan(&pk.Reason)
	return pk, err
}
