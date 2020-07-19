package main

import (
	"github.com/Velocity-/gorune"
	"github.com/Velocity-/microlog"
	"net"
	"bufio"
	"encoding/binary"
	"hash/crc32"
)

var globalDescriptor []byte

func StartServer(fs *gorune.FileSystem, revision int) {
	addr, _ := net.ResolveTCPAddr("tcp4", "127.0.0.1:43594")
	listener, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		panic(err)
	}

	// Precompute global descriptor
	microlog.Info("Computing descriptor data...")
	globalDescriptor = trim(getGlobalDescriptor(fs))

	microlog.Info("Accepting connections.")
	for {
		con, err := listener.AcceptTCP()

		if err != nil {
			microlog.Error("Error accepting tcp connection: %s", err)
		} else {
			microlog.Info("New connection: %s", con.RemoteAddr().String())

			go handleConnection(fs, con, revision)
		}
	}
}

func handleConnection(fs *gorune.FileSystem, con *net.TCPConn, revision int) {
	reader := bufio.NewReader(con)
	temp := make([]byte, 4)

	for {
		opcode, err := reader.ReadByte()
		if err != nil {
			break // Connection assumed dead
		}

		if opcode == 15 {
			reader.Read(temp[:4])
			clientRevision := int(binary.BigEndian.Uint32(temp))

			if revision == clientRevision {
				con.Write([]byte{0}) // Connection accepted
			} else {
				con.Write([]byte{6}) // Connection denied - out of date.
				con.Close()
				return
			}
		} else if opcode == 2 || opcode == 3 || opcode == 6 {
			reader.Read(temp[:3]) // Data is ignored
		} else if opcode == 0 || opcode == 1 {
			reader.Read(temp[:3])
			index := int(temp[0] & 0xFF)
			file := int(binary.BigEndian.Uint16(temp[1:]))

			if index == 255 && file == 255 {
				writeChunked(con, 255, 255, opcode == 0, globalDescriptor)
			} else {
				data, err := fs.ReadRaw(fs.Indices[index].Entries[file])
				if err == nil {
					writeChunked(con, index, file, opcode == 0, trim(data))
				}
			}
		}
	}
}

func getGlobalDescriptor(fs *gorune.FileSystem) []byte {
	resp := make([]byte, 32 + 17 * 8)

	resp[0] = 0 // Compression
	binary.BigEndian.PutUint32(resp[1:], 17 * 8) // Size
	pos := 5

	for index := 0; index < 17; index++ {
		rawData, _ := fs.ReadRaw(fs.Indices[255].Entries[index])
		data, _ := fs.ReadDecompressed(fs.Indices[255].Entries[index])
		table, _ := gorune.DecodeReferenceTable(data)

		binary.BigEndian.PutUint32(resp[pos:], crc32.Checksum(rawData, crc32.IEEETable))
		binary.BigEndian.PutUint32(resp[pos+4:], uint32(table.Revision))

		pos += 8
	}

	return resp[0 : pos]
}

func trim(data []byte) []byte {
	compression := data[0]
	size := binary.BigEndian.Uint32(data[1:])

	headerLength := uint32(5)
	if compression != 0 {
		headerLength = uint32(9)
	}

	return data[0 : size+headerLength]
}

func writeChunked(con *net.TCPConn, index int, entryId int, urgent bool, data []byte) {
	// Write the header
	tmp := make([]byte, 3)
	tmp[0] = byte(index)
	binary.BigEndian.PutUint16(tmp[1:], uint16(entryId))
	con.Write(tmp[0:3])

	position := 3
	srcpos := 0

	for remaining := len(data); remaining > 0; {
		// A block can only contain 512 bytes
		blockLen := 512 - (position % 512)
		if blockLen > remaining {
			blockLen = remaining
		}

		// Write block data
		con.Write(data[srcpos : srcpos+blockLen])
		position += blockLen
		srcpos += blockLen
		remaining -= blockLen

		// Mark a chunk after 512 bytes with a negative (255 or -1) byte.
		if position%512 == 0 && remaining > 0 {
			con.Write([]byte{255})
			position++
		}
	}
}
