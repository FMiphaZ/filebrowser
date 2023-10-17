package services

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	Afc_magic       uint64 = 0x4141504c36414643
	Afc_header_size uint64 = 40

	Afc_operation_status              uint64 = 0x00000001
	Afc_operation_data                uint64 = 0x00000002 // Data
	Afc_operation_read_dir            uint64 = 0x00000003 // ReadDir
	Afc_operation_READ_FILE           uint64 = 0x00000004 // ReadFile
	Afc_operation_WRITE_FILE          uint64 = 0x00000005 // WriteFile
	Afc_operation_WRITE_PART          uint64 = 0x00000006 // WritePart
	Afc_operation_TRUNCATE            uint64 = 0x00000007 // TruncateFile
	Afc_operation_remove_path         uint64 = 0x00000008 // RemovePath
	Afc_operation_make_dir            uint64 = 0x00000009 // MakeDir
	Afc_operation_file_info           uint64 = 0x0000000A // GetFileInfo
	Afc_operation_get_devinfo         uint64 = 0x0000000B // GetDeviceInfo
	Afc_operation_write_file_atom     uint64 = 0x0000000C // WriteFileAtomic (tmp file+rename)
	Afc_operation_file_open           uint64 = 0x0000000D // FileRefOpen
	Afc_operation_file_open_result    uint64 = 0x0000000E // FileRefOpenResult
	Afc_operation_file_read           uint64 = 0x0000000F // FileRefRead
	Afc_operation_file_write          uint64 = 0x00000010 // FileRefWrite
	Afc_operation_file_seek           uint64 = 0x00000011 // FileRefSeek
	Afc_operation_file_tell           uint64 = 0x00000012 // FileRefTell
	Afc_operation_file_tell_result    uint64 = 0x00000013 // FileRefTellResult
	Afc_operation_file_close          uint64 = 0x00000014 // FileRefClose
	Afc_operation_file_set_size       uint64 = 0x00000015 // FileRefSetFileSize(ftruncate)
	Afc_operation_get_con_info        uint64 = 0x00000016 // GetConnectionInfo
	Afc_operation_set_conn_options    uint64 = 0x00000017 // SetConnectionOptions
	Afc_operation_rename_path         uint64 = 0x00000018 // RenamePath
	Afc_operation_set_fs_bs           uint64 = 0x00000019 // SetFSBlockSize (0x800000)
	Afc_operation_set_socket_bs       uint64 = 0x0000001A // SetSocketBlockSize
	Afc_operation_file_lock           uint64 = 0x0000001B // FileRefLock
	Afc_operation_make_link           uint64 = 0x0000001C // MakeLink
	Afc_operation_set_file_time       uint64 = 0x0000001E // set st_mtime
	Afc_operation_get_file_Hash_range uint64 = 0x0000001F // GetFileHashWithRange

	/* iOS 6+ */
	AFC_OP_FILE_SET_IMMUTABLE_HINT   = 0x00000020 /* FileRefSetImmutableHint */
	AFC_OP_GET_SIZE_OF_PATH_CONTENTS = 0x00000021 /* GetSizeOfPathContents */
	AFC_OP_REMOVE_PATH_AND_CONTENTS  = 0x00000022 /* RemovePathAndContents */
	AFC_OP_DIR_OPEN                  = 0x00000023 /* DirectoryEnumeratorRefOpen */
	AFC_OP_DIR_OPEN_RESULT           = 0x00000024 /* DirectoryEnumeratorRefOpenResult */
	AFC_OP_DIR_READ                  = 0x00000025 /* DirectoryEnumeratorRefRead */
	AFC_OP_DIR_CLOSE                 = 0x00000026 /* DirectoryEnumeratorRefClose */
	/* iOS 7+ */
	AFC_OP_FILE_READ_OFFSET  = 0x00000027 /* FileRefReadWithOffset */
	AFC_OP_FILE_WRITE_OFFSET = 0x00000028 /* FileRefWriteWithOffset */
)

type LinkType int

const (
	AFC_HARDLINK LinkType = 1
	AFC_SYMLINK  LinkType = 2
)

const (
	Afc_Mode_RDONLY   uint64 = 0x00000001 // r,  O_RDONLY
	Afc_Mode_RW       uint64 = 0x00000002 // r+, O_RDWR   | O_CREAT
	Afc_Mode_WRONLY   uint64 = 0x00000003 // w,  O_WRONLY | O_CREAT  | O_TRUNC
	Afc_Mode_WR       uint64 = 0x00000004 // w+, O_RDWR   | O_CREAT  | O_TRUNC
	Afc_Mode_APPEND   uint64 = 0x00000005 // a,  O_WRONLY | O_APPEND | O_CREAT
	Afc_Mode_RDAPPEND uint64 = 0x00000006 // a+, O_RDWR   | O_APPEND | O_CREAT
)

type AfcErr uint64

const (
	Afc_Err_Success                = AfcErr(0)
	Afc_Err_UnknownError           = AfcErr(1)
	Afc_Err_OperationHeaderInvalid = AfcErr(2)
	Afc_Err_NoResources            = AfcErr(3)
	Afc_Err_ReadError              = AfcErr(4)
	Afc_Err_WriteError             = AfcErr(5)
	Afc_Err_UnknownPacketType      = AfcErr(6)
	Afc_Err_InvalidArgument        = AfcErr(7)
	Afc_Err_ObjectNotFound         = AfcErr(8)
	Afc_Err_ObjectIsDir            = AfcErr(9)
	Afc_Err_PermDenied             = AfcErr(10)
	Afc_Err_ServiceNotConnected    = AfcErr(11)
	Afc_Err_OperationTimeout       = AfcErr(12)
	Afc_Err_TooMuchData            = AfcErr(13)
	Afc_Err_EndOfData              = AfcErr(14)
	Afc_Err_OperationNotSupported  = AfcErr(15)
	Afc_Err_ObjectExists           = AfcErr(16)
	Afc_Err_ObjectBusy             = AfcErr(17)
	Afc_Err_NoSpaceLeft            = AfcErr(18)
	Afc_Err_OperationWouldBlock    = AfcErr(19)
	Afc_Err_IoError                = AfcErr(20)
	Afc_Err_OperationInterrupted   = AfcErr(21)
	Afc_Err_OperationInProgress    = AfcErr(22)
	Afc_Err_InternalError          = AfcErr(23)
	Afc_Err_MuxError               = AfcErr(30)
	Afc_Err_NoMemory               = AfcErr(31)
	Afc_Err_NotEnoughData          = AfcErr(32)
	Afc_Err_DirNotEmpty            = AfcErr(33)
)

func (errorCode AfcErr) Error() error {
	switch errorCode {
	case Afc_Err_Success:
		return nil
	case Afc_Err_UnknownError:
		return errors.New("UnknownError")
	case Afc_Err_OperationHeaderInvalid:
		return errors.New("OperationHeaderInvalid")
	case Afc_Err_NoResources:
		return errors.New("NoResources")
	case Afc_Err_ReadError:
		return errors.New("ReadError")
	case Afc_Err_WriteError:
		return errors.New("WriteError")
	case Afc_Err_UnknownPacketType:
		return errors.New("UnknownPacketType")
	case Afc_Err_InvalidArgument:
		return errors.New("InvalidArgument")
	case Afc_Err_ObjectNotFound:
		return errors.New("ObjectNotFound")
	case Afc_Err_ObjectIsDir:
		return errors.New("ObjectIsDir")
	case Afc_Err_PermDenied:
		return errors.New("PermDenied")
	case Afc_Err_ServiceNotConnected:
		return errors.New("ServiceNotConnected")
	case Afc_Err_OperationTimeout:
		return errors.New("OperationTimeout")
	case Afc_Err_TooMuchData:
		return errors.New("TooMuchData")
	case Afc_Err_EndOfData:
		return errors.New("EndOfData")
	case Afc_Err_OperationNotSupported:
		return errors.New("OperationNotSupported")
	case Afc_Err_ObjectExists:
		return errors.New("ObjectExists")
	case Afc_Err_ObjectBusy:
		return errors.New("ObjectBusy")
	case Afc_Err_NoSpaceLeft:
		return errors.New("NoSpaceLeft")
	case Afc_Err_OperationWouldBlock:
		return errors.New("OperationWouldBlock")
	case Afc_Err_IoError:
		return errors.New("IoError")
	case Afc_Err_OperationInterrupted:
		return errors.New("OperationInterrupted")
	case Afc_Err_OperationInProgress:
		return errors.New("OperationInProgress")
	case Afc_Err_InternalError:
		return errors.New("InternalError")
	case Afc_Err_MuxError:
		return errors.New("MuxError")
	case Afc_Err_NoMemory:
		return errors.New("NoMemory")
	case Afc_Err_NotEnoughData:
		return errors.New("NotEnoughData")
	case Afc_Err_DirNotEmpty:
		return errors.New("DirNotEmpty")
	default:
		return nil
	}
}

type AfcPacketHeader struct {
	Magic         uint64
	Entire_length uint64
	This_length   uint64
	Packet_num    uint64
	Operation     uint64
}

type AfcPacket struct {
	Header        AfcPacketHeader
	HeaderPayload []byte
	Payload       []byte
}

func (p *AfcPacket) Pack() []byte {
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.LittleEndian, p.Header)
	buf.Write(p.HeaderPayload)
	buf.Write(p.Payload)
	return buf.Bytes()
}

func (p *AfcPacket) PackTo(writer io.Writer) error {
	err := binary.Write(writer, binary.LittleEndian, p.Header)
	if err != nil {
		return err
	}
	_, err = writer.Write(p.HeaderPayload)
	if err != nil {
		return err
	}

	_, err = writer.Write(p.Payload)
	return err
}

func (p *AfcPacket) Error() error {
	if p.Header.Operation == Afc_operation_status {
		errorCode := AfcErr(binary.LittleEndian.Uint64(p.HeaderPayload))
		return errorCode.Error()
	}
	return nil
}

func UnpackAfcPacket(reader io.Reader) (p AfcPacket, err error) {
	var header AfcPacketHeader
	err = binary.Read(reader, binary.LittleEndian, &header)
	if err != nil {
		return
	}
	if header.Magic != Afc_magic {
		err = fmt.Errorf("wrong magic:%x expected: %x", header.Magic, Afc_magic)
		return
	}
	headerPayloadLength := header.This_length - Afc_header_size
	headerPayload := make([]byte, headerPayloadLength)
	_, err = io.ReadFull(reader, headerPayload)
	if err != nil {
		return
	}
	contentPayloadLength := header.Entire_length - header.This_length
	payload := make([]byte, contentPayloadLength)
	_, err = io.ReadFull(reader, payload)
	if err != nil {
		return
	}
	p = AfcPacket{header, headerPayload, payload}
	return
}

type StatInfo struct {
	name         string
	stSize       int64
	stBlocks     int64
	stCtime      int64
	stMtime      int64
	stNlink      string
	stIfmt       string
	stLinktarget string
}

func (s *StatInfo) Name() string {
	return s.name
}

func (s *StatInfo) Size() int64 {
	return s.stSize
}

func (s *StatInfo) Mode() os.FileMode {
	if s.stIfmt == "S_IFDIR" {
		return os.ModeDir
	}
	return 0
}

func (s *StatInfo) CTime() time.Time {
	return time.UnixMicro(s.stCtime / 1000)
}

func (s *StatInfo) ModTime() time.Time {
	return time.UnixMicro(s.stMtime / 1000)
}

func (s *StatInfo) Sys() interface{} {
	return s
}

func (s *StatInfo) IsDir() bool {
	return s.stIfmt == "S_IFDIR"
}

func (s *StatInfo) IsLink() bool {
	return s.stIfmt == "S_IFLNK"
}

func (s *StatInfo) SetName(name string) *StatInfo {
	s.name = name
	return s
}

func (s *StatInfo) SetTime(stCtime, stMtime time.Time) *StatInfo {
	s.stCtime = stCtime.UnixNano()
	s.stMtime = stMtime.UnixNano()
	return s
}

func NewDirStatInfo(name string) *StatInfo {
	return &StatInfo{
		name:         path.Base(name),
		stSize:       0,
		stBlocks:     0,
		stCtime:      time.Now().UnixNano(),
		stMtime:      time.Now().UnixNano(),
		stNlink:      "",
		stIfmt:       "S_IFDIR",
		stLinktarget: "",
	}
}

type AfcService struct {
	Conn          net.Conn
	addr          string
	packageNumber uint64
	mutex         sync.Mutex
}

func NewAfcService(addr string) (*AfcService, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	s := &AfcService{
		addr: addr,
		Conn: conn,
	}
	return s, err
}

func (conn *AfcService) request(ops uint64, data, payload []byte) (*AfcPacket, error) {
	header := AfcPacketHeader{
		Magic:         Afc_magic,
		Packet_num:    conn.packageNumber,
		Operation:     ops,
		This_length:   Afc_header_size + uint64(len(data)),
		Entire_length: Afc_header_size + uint64(len(data)+len(payload)),
	}

	packet := AfcPacket{
		Header:        header,
		HeaderPayload: data,
		Payload:       payload,
	}

	conn.packageNumber++
	// 这里有两种写法
	_, err := conn.Conn.Write(packet.Pack())
	// err := packet.PackTo(conn.Conn)
	if err != nil {
		return nil, err
	}
	response, err := UnpackAfcPacket(conn.Conn)
	if err != nil {
		return nil, err
	}

	err = response.Error()
	return &response, err
}

func (conn *AfcService) RemovePath(path string) error {
	log.Debugf("Remove path %v", path)
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	_, err := conn.request(Afc_operation_remove_path, []byte(path), nil)
	return err
}

func (conn *AfcService) RenamePath(from, to string) error {
	data := make([]byte, len(from)+1+len(to)+1)
	copy(data, from)
	copy(data[len(from)+1:], to)
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	_, err := conn.request(Afc_operation_rename_path, data, nil)
	return err
}

func (conn *AfcService) MakeDir(path string) error {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	_, err := conn.request(Afc_operation_make_dir, []byte(path), nil)
	return err
}

func (conn *AfcService) Stat(path string) (*StatInfo, error) {
	conn.mutex.Lock()
	response, err := conn.request(Afc_operation_file_info, []byte(path), nil)
	if err != nil {
		conn.mutex.Unlock()
		return nil, fmt.Errorf("cannot stat '%v': %v", path, err)
	}
	conn.mutex.Unlock()

	ret := bytes.Split(bytes.TrimSuffix(response.Payload, []byte{0}), []byte{0})
	if len(ret)%2 != 0 {
		log.Fatalf("invalid response: %v %% 2 != 0", len(ret))
	}

	statInfoMap := make(map[string]string)
	for i := 0; i < len(ret); i = i + 2 {
		k := string(ret[i])
		v := string(ret[i+1])
		statInfoMap[k] = v
	}

	var si StatInfo
	si.name = filepath.Base(path)
	si.stSize, _ = strconv.ParseInt(statInfoMap["st_size"], 10, 64)
	si.stBlocks, _ = strconv.ParseInt(statInfoMap["st_blocks"], 10, 64)
	si.stCtime, _ = strconv.ParseInt(statInfoMap["st_birthtime"], 10, 64)
	si.stMtime, _ = strconv.ParseInt(statInfoMap["st_mtime"], 10, 64)
	si.stNlink = statInfoMap["st_nlink"]
	si.stIfmt = statInfoMap["st_ifmt"]
	si.stLinktarget = statInfoMap["st_linktarget"]
	return &si, nil
}

func (conn *AfcService) ReadDir(path string) ([]string, error) {
	//log.Debugf("ReadDir path:%v", path)
	conn.mutex.Lock()
	response, err := conn.request(Afc_operation_read_dir, []byte(path), nil)
	if err != nil {
		conn.mutex.Unlock()
		log.Infof("ReadDir error:%v", err)
		return nil, err
	}
	conn.mutex.Unlock()

	ret := bytes.Split(bytes.TrimSuffix(response.Payload, []byte{0}), []byte{0})
	var fileList []string
	for _, v := range ret {
		if string(v) != "." && string(v) != ".." && string(v) != "" {
			fileList = append(fileList, string(v))
		}
	}

	//log.Debugf("ReadDir end:%v", fileList)
	return fileList, nil
}

func (conn *AfcService) OpenFile(path string, mode uint64) (uint64, error) {
	log.Debugf("OpenFile path:%v", path)
	data := make([]byte, 8+len(path)+1)
	binary.LittleEndian.PutUint64(data, mode)
	copy(data[8:], path)
	conn.mutex.Lock()
	response, err := conn.request(Afc_operation_file_open, data, make([]byte, 0))
	if err != nil {
		conn.mutex.Unlock()
		log.Errorf("OpenFile path:%v err:%v", path, err)
		return 0, err
	}
	conn.mutex.Unlock()

	fd := binary.LittleEndian.Uint64(response.HeaderPayload)
	if fd == 0 {
		return 0, fmt.Errorf("file descriptor should not be zero")
	}

	return fd, nil
}

func (conn *AfcService) ReadFile(fd uint64, p []byte) (n int, err error) {
	log.Debugf("ReadFile inbuf pd:%v, read len:%v", fd, len(p))
	defer log.Info("ReadFile end")
	data := make([]byte, 16)
	binary.LittleEndian.PutUint64(data, fd)
	binary.LittleEndian.PutUint64(data[8:], uint64(len(p)))

	conn.mutex.Lock()
	response, err := conn.request(Afc_operation_file_read, data, nil)
	if err != nil {
		conn.mutex.Unlock()
		return 0, err
	}
	conn.mutex.Unlock()

	log.Debugf("inbuf len:%v, read len:%v", len(p), len(response.Payload))
	n = len(response.Payload)
	if n > len(p) {
		log.Fatalf("inbuf len:%v, read len:%v", len(p), len(response.Payload))
	}
	if n == 0 {
		return n, io.EOF
	}

	if n < len(p) {
		err = io.EOF
	}
	copy(p, response.Payload)
	return
}

func (conn *AfcService) WriteFile(fd uint64, p []byte) (n int, err error) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, fd)

	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	_, err = conn.request(Afc_operation_file_write, data, p)
	return len(p), err
}

func (conn *AfcService) CloseFile(fd uint64) error {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, fd)

	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	_, err := conn.request(Afc_operation_file_close, data, nil)
	return err
}

func (conn *AfcService) LockFile(fd uint64) error {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, fd)

	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	_, err := conn.request(Afc_operation_file_close, data, nil)
	return err
}

// SeekFile whence is SEEK_SET, SEEK_CUR, or SEEK_END.
func (conn *AfcService) SeekFile(fd uint64, offset int64, whence int) (int64, error) {
	data := make([]byte, 24)
	binary.LittleEndian.PutUint64(data, fd)
	binary.LittleEndian.PutUint64(data[8:], uint64(whence))
	binary.LittleEndian.PutUint64(data[16:], uint64(offset))

	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	_, err := conn.request(Afc_operation_file_seek, data, nil)
	if err != nil {
		return 0, err
	}

	data2 := make([]byte, 8)
	binary.LittleEndian.PutUint64(data2, fd)
	response, err := conn.request(Afc_operation_file_tell, data2, nil)
	if err != nil {
		return 0, err
	}

	pos := binary.LittleEndian.Uint64(response.HeaderPayload)
	return int64(pos), nil
}

func (conn *AfcService) TellFile(fd uint64) (uint64, error) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, fd)

	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	response, err := conn.request(Afc_operation_file_tell, data, nil)
	if err != nil {
		return 0, err
	}

	pos := binary.LittleEndian.Uint64(response.HeaderPayload)
	return pos, err
}

func (conn *AfcService) TruncateFile(fd uint64, size int64) error {
	data := make([]byte, 16)
	binary.LittleEndian.PutUint64(data, fd)
	binary.LittleEndian.PutUint64(data[8:], uint64(size))

	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	_, err := conn.request(Afc_operation_file_set_size, data, nil)
	return err
}

func (conn *AfcService) Truncate(path string, size uint64) error {
	data := make([]byte, 8+len(path))
	binary.LittleEndian.PutUint64(data, size)
	copy(data[8:], path)

	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	_, err := conn.request(Afc_operation_TRUNCATE, data, nil)
	return err
}

func (conn *AfcService) MakeLink(link LinkType, target, linkname string) error {
	data := make([]byte, 8+len(target)+1+len(linkname)+1)
	binary.LittleEndian.PutUint64(data, uint64(link))
	copy(data[8:], target)
	copy(data[8+len(target)+1:], linkname)

	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	_, err := conn.request(Afc_operation_make_link, data, nil)
	return err
}

func (conn *AfcService) SetFileTime(path string, t time.Time) error {
	data := make([]byte, 8+len(path)+1)
	binary.LittleEndian.PutUint64(data, uint64(t.UnixNano()))
	copy(data[8:], path)

	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	_, err := conn.request(Afc_operation_set_file_time, data, nil)
	return err
}

func (conn *AfcService) RemovePathAndContents(path string) error {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	_, err := conn.request(AFC_OP_REMOVE_PATH_AND_CONTENTS, []byte(path), nil)
	return err
}

func (conn *AfcService) Close() error {
	return conn.Conn.Close()
}
