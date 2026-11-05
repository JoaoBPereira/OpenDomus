package devices

import (
	"bytes"
	"crypto/aes"
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net"
	"strconv"
	"time"
)

// Tuya talks the Tuya local protocol version 3.3, replacing the tuyapi npm package:
// the device IP is discovered with the UDP broadcasts the devices send every few seconds, then
// AES-128-ECB encrypted frames are exchanged over TCP port 6668.
//
// Frame layout: 000055AA | sequence (4) | command (4) | length (4) | payload | crc32 (4) | 0000AA55
// Note: ported from the tuyapi sources, not yet tested against a real device.
type Tuya struct {
	ID  string
	Key string
	IP  string

	connection net.Conn
	sequence   uint32
}

const tuyaTCPPort = "6668"
const tuyaUDPPort = "6666"          // plain broadcasts (protocol 3.1)
const tuyaUDPEncryptedPort = "6667" // encrypted broadcasts (protocol 3.3)
const tuyaCommandDpQuery = 0x0a
const tuyaFindTimeout = 10 * time.Second
const tuyaRequestTimeout = 10 * time.Second

var tuyaPrefix = []byte{0x00, 0x00, 0x55, 0xaa}
var tuyaSuffix = []byte{0x00, 0x00, 0xaa, 0x55}
var tuyaUDPKey = md5.Sum([]byte("yGAdlopoPVldABfn"))

func NewTuya(id string, key string) *Tuya {
	return &Tuya{ID: id, Key: key}
}

// Find discovers the IP of the device from its UDP broadcasts. Skipped when IP is already known.
func (device *Tuya) Find() error {
	if device.IP != "" {
		return nil
	}

	found := make(chan string, 2)
	stop := make(chan struct{})
	defer close(stop)
	go device.listenForBroadcasts(tuyaUDPPort, false, found, stop)
	go device.listenForBroadcasts(tuyaUDPEncryptedPort, true, found, stop)

	select {
	case ip := <-found:
		device.IP = ip
		return nil
	case <-time.After(tuyaFindTimeout):
		return errors.New("find() timed out. Is the device powered on and on the same network?")
	}
}

func (device *Tuya) listenForBroadcasts(port string, encrypted bool, found chan<- string, stop <-chan struct{}) {
	address, err := net.ResolveUDPAddr("udp4", ":"+port)
	if err != nil {
		return
	}
	socket, err := net.ListenUDP("udp4", address)
	if err != nil {
		return
	}
	defer socket.Close()
	go func() {
		<-stop
		socket.Close()
	}()

	buffer := make([]byte, 2048)
	for {
		socket.SetReadDeadline(time.Now().Add(tuyaFindTimeout))
		length, _, err := socket.ReadFromUDP(buffer)
		if err != nil {
			return
		}
		payload, err := tuyaDecodeFrame(buffer[:length])
		if err != nil {
			continue
		}
		if encrypted {
			payload, err = tuyaDecrypt(payload, tuyaUDPKey[:])
			if err != nil {
				continue
			}
		}
		var broadcast struct {
			IP   string `json:"ip"`
			GwID string `json:"gwId"`
		}
		if json.Unmarshal(payload, &broadcast) != nil {
			continue
		}
		if broadcast.GwID == device.ID && broadcast.IP != "" {
			found <- broadcast.IP
			return
		}
	}
}

func (device *Tuya) Connect() error {
	if device.IP == "" {
		return errors.New("device IP is unknown, call Find() first")
	}
	connection, err := net.DialTimeout("tcp", net.JoinHostPort(device.IP, tuyaTCPPort), tuyaRequestTimeout)
	if err != nil {
		return err
	}
	device.connection = connection
	return nil
}

func (device *Tuya) Disconnect() {
	if device.connection != nil {
		device.connection.Close()
		device.connection = nil
	}
}

// Get asks the device for all its data points (DP_QUERY) and returns the one with the given number
func (device *Tuya) Get(dps int) (any, error) {
	if device.connection == nil {
		return nil, errors.New("not connected")
	}

	request := map[string]any{
		"gwId":  device.ID,
		"devId": device.ID,
		"t":     strconv.FormatInt(time.Now().Unix(), 10),
		"dps":   map[string]any{},
		"uid":   device.ID,
	}
	requestJSON, _ := json.Marshal(request)
	encrypted, err := tuyaEncrypt(requestJSON, []byte(device.Key))
	if err != nil {
		return nil, err
	}

	device.connection.SetDeadline(time.Now().Add(tuyaRequestTimeout))
	if _, err := device.connection.Write(device.encodeFrame(tuyaCommandDpQuery, encrypted)); err != nil {
		return nil, err
	}

	// The device may send other frames first (heartbeats, status reports): wait for the DP_QUERY answer
	var payload []byte
	for attempt := 0; attempt < 5; attempt++ {
		frame, err := device.readFrame()
		if err != nil {
			return nil, err
		}
		command := binary.BigEndian.Uint32(frame[8:12])
		if command == tuyaCommandDpQuery {
			payload, err = tuyaDecodeFrame(frame)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	if payload == nil {
		return nil, errors.New("no DP_QUERY answer from the device")
	}

	// Answers are normally encrypted, a few devices answer in plain text
	data := payload
	if decrypted, err := tuyaDecrypt(payload, []byte(device.Key)); err == nil {
		data = decrypted
	}
	var response struct {
		Dps map[string]any `json:"dps"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("could not parse the device answer: %w", err)
	}
	value, exists := response.Dps[strconv.Itoa(dps)]
	if !exists {
		return nil, fmt.Errorf("dps %d not found in the device answer", dps)
	}
	return value, nil
}

func (device *Tuya) encodeFrame(command uint32, payload []byte) []byte {
	device.sequence++
	frame := make([]byte, 0, 16+len(payload)+8)
	frame = append(frame, tuyaPrefix...)
	frame = binary.BigEndian.AppendUint32(frame, device.sequence)
	frame = binary.BigEndian.AppendUint32(frame, command)
	frame = binary.BigEndian.AppendUint32(frame, uint32(len(payload)+8))
	frame = append(frame, payload...)
	frame = binary.BigEndian.AppendUint32(frame, crc32.ChecksumIEEE(frame))
	frame = append(frame, tuyaSuffix...)
	return frame
}

// readFrame reads one complete frame from the TCP connection
func (device *Tuya) readFrame() ([]byte, error) {
	header := make([]byte, 16)
	if _, err := io.ReadFull(device.connection, header); err != nil {
		return nil, err
	}
	if !bytes.Equal(header[0:4], tuyaPrefix) {
		return nil, errors.New("not a Tuya frame")
	}
	payloadLength := int(binary.BigEndian.Uint32(header[12:16]))
	if payloadLength > 65536 {
		return nil, errors.New("Tuya frame too large")
	}
	rest := make([]byte, payloadLength)
	if _, err := io.ReadFull(device.connection, rest); err != nil {
		return nil, err
	}
	return append(header, rest...), nil
}

// tuyaDecodeFrame validates a frame (prefix, length, crc, suffix) and returns its payload,
// without the return code and the "3.3" version header the device may put in front of it
func tuyaDecodeFrame(frame []byte) ([]byte, error) {
	if len(frame) < 24 || !bytes.Equal(frame[0:4], tuyaPrefix) {
		return nil, errors.New("not a Tuya frame")
	}
	payloadLength := int(binary.BigEndian.Uint32(frame[12:16]))
	frameEnd := 16 + payloadLength
	if payloadLength < 8 || frameEnd > len(frame) {
		return nil, errors.New("truncated Tuya frame")
	}
	if !bytes.Equal(frame[frameEnd-4:frameEnd], tuyaSuffix) {
		return nil, errors.New("Tuya frame suffix mismatch")
	}
	expectedCRC := binary.BigEndian.Uint32(frame[frameEnd-8 : frameEnd-4])
	if crc32.ChecksumIEEE(frame[0:frameEnd-8]) != expectedCRC {
		return nil, errors.New("Tuya frame crc mismatch")
	}

	payload := frame[16 : frameEnd-8]
	// Answers start with a 4 byte return code (0 = ok)
	if len(payload) >= 4 && binary.BigEndian.Uint32(payload[0:4])&0xFFFFFF00 == 0 {
		payload = payload[4:]
	}
	// Protocol 3.3 may prefix the payload with "3.3" and 12 header bytes
	if bytes.HasPrefix(payload, []byte("3.3")) && len(payload) >= 15 {
		payload = payload[15:]
	}
	return payload, nil
}

// tuyaEncrypt is AES-128-ECB with PKCS#7 padding
func tuyaEncrypt(plain []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	padding := aes.BlockSize - len(plain)%aes.BlockSize
	padded := append(append([]byte{}, plain...), bytes.Repeat([]byte{byte(padding)}, padding)...)
	encrypted := make([]byte, len(padded))
	for start := 0; start < len(padded); start += aes.BlockSize {
		block.Encrypt(encrypted[start:start+aes.BlockSize], padded[start:start+aes.BlockSize])
	}
	return encrypted, nil
}

func tuyaDecrypt(encrypted []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(encrypted) == 0 || len(encrypted)%aes.BlockSize != 0 {
		return nil, errors.New("encrypted data is not a multiple of the AES block size")
	}
	decrypted := make([]byte, len(encrypted))
	for start := 0; start < len(encrypted); start += aes.BlockSize {
		block.Decrypt(decrypted[start:start+aes.BlockSize], encrypted[start:start+aes.BlockSize])
	}
	padding := int(decrypted[len(decrypted)-1])
	if padding < 1 || padding > aes.BlockSize || padding > len(decrypted) {
		return nil, errors.New("invalid padding in the decrypted data")
	}
	return decrypted[:len(decrypted)-padding], nil
}
