package spdy

import (
	"bytes"
	"fmt"
)

const Version = 2

const (
	SYN_STREAM uint16 = iota + 1
	SYN_REPLY
	RST_STREAM
	SETTINGS
	NOOP
	PING
	GOAWAY
	HEADERS
)
const (
	FLAG_FIN            uint8 = 0x01
	FLAG_UNIDIRECTIONAL       = 0x02
)

type Frame interface {
	Len() uint32
}

/*

Control frames

+----------------------------------+
|C| Version(15bits) | Type(16bits) |
+----------------------------------+
| Flags (8)  |  Length (24 bits)   |
+----------------------------------+
|               Data               |
+----------------------------------+

*/
type CtrlFrameHead struct {
	Frame
	Version uint16 // 15 bits
	Type    uint16
	Flags   uint8
	Length  uint32
}

func (h *CtrlFrameHead) Len() uint32 {
	return h.Length
}

func (h *CtrlFrameHead) Head() string {
	return fmt.Sprintf("CtrlFrameHead{Version=%d, Type=%d, Flags=%d, Length=%d}",
		h.Version, h.Type, h.Flags, h.Length)
}

/*

Data frames

+----------------------------------+
|C|       Stream-ID (31bits)       |
+----------------------------------+
| Flags (8)  |  Length (24 bits)   |
+----------------------------------+
|               Data               |
+----------------------------------+

*/
type DataFrame struct {
	StreamId uint32
	Flags    uint8
	Length   uint32

	Data *bytes.Buffer
}

func (f *DataFrame) Len() uint32 {
	return f.Length
}

func (h *DataFrame) Head() string {
	return fmt.Sprintf("DataFrameHead{StreamId=%d, Flags=%d, Length=%d}",
		h.StreamId, h.Flags, h.Length)
}

/*

SYN_STREAM

+----------------------------------+
|1|       2          |       1     |
+----------------------------------+
| Flags (8)  |  Length (24 bits)   |
+----------------------------------+
|X|          Stream-ID (31bits)    |
+----------------------------------+
|X|Associated-To-Stream-ID (31bits)|
+----------------------------------+
|Pri(2)|Unused(14)|                |
+-----------------+                |
|     Name/value header block      |
|             ...                  |
+----------------------------------+


Name/Value header block format


+------------------------------------+
| Number of Name/Value pairs (int16) |
+------------------------------------+
|     Length of name (int16)         |
+------------------------------------+
|           Name (string)            |
+------------------------------------+
|     Length of value  (int16)       |
+------------------------------------+
|          Value   (string)          |
+------------------------------------+
|           (repeats)                |
+------------------------------------+

*/
type SynStreamFrame struct {
	CtrlFrameHead

	StreamId     uint32
	AssociatedId uint32
	Priority     uint16 // Priority: A 2-bit priority field

	Header map[string]string
}

func NewSynStreamFrame(streamId uint32) *SynStreamFrame {
	frame := &SynStreamFrame{
		CtrlFrameHead: CtrlFrameHead{
			Version: Version,
			Type:    1,
		},
		StreamId:     streamId,
		AssociatedId: 0,
		Priority:     3,
		Header:       make(map[string]string),
	}

	return frame
}

func (syn *SynStreamFrame) String() string {
	return fmt.Sprintf("SynStreamFrame{"+
		"Ctrl: true, Version: %d, Type: %d, Flags: %d, Length: %d, "+
		"StreamId: %d, AssociatedId: %d, Priority: %d, Header: %v }",
		syn.Version, syn.Type, syn.Flags, syn.Length, syn.StreamId,
		syn.AssociatedId, syn.Priority<<14, syn.Header)
}

/*

SYN_REPLY

+----------------------------------+
|1|        2        |        2     |
+----------------------------------+
| Flags (8)  |  Length (24 bits)   |
+----------------------------------+
|X|          Stream-ID (31bits)    |
+----------------------------------+
| Unused        |                  |
+----------------                  |
|     Name/value header block      |
|              ...                 |
+----------------------------------+

*/
type SynReplyFrame struct {
	CtrlFrameHead

	StreamId uint32

	Header map[string]string
}

/*

RST_STREAM

+-------------------------------+
|1|       2        |      3     |
+-------------------------------+
| Flags (8)  |         8        |
+-------------------------------+
|X|          Stream-ID (31bits) |
+-------------------------------+
|          Status code          |
+-------------------------------+
*/
type RstStreamFrame struct {
	CtrlFrameHead
	StreamId uint32
	Status   uint32
}

/*

SETTINGS

+----------------------------------+
|1|       2          |       4     |
+----------------------------------+
| Flags (8)  |  Length (24 bits)   |
+----------------------------------+
|   Number of entries (32 bits)    |
+----------------------------------+
|          ID/Value Pairs          |
|             ...                  |
+----------------------------------+

Each ID/value pair is as follows:
+----------------------------------+
|    ID (24 bits)   | ID_Flags (8) |
+----------------------------------+
|          Value (32 bits)         |
+----------------------------------+

*/

type SettingsFrame struct {
	CtrlFrameHead

	Settings []Setting
}
type Setting struct {
	Id    uint32
	Flag  uint8
	Value uint32
}

/*

NOOP

+----------------------------------+
|1|       2          |       5     |
+----------------------------------+
| 0 (Flags)  |    0 (Length)       |
+----------------------------------+

*/
type NoopFrame struct {
	CtrlFrameHead
}

/*

PING

+----------------------------------+
|1|       2          |       6     |
+----------------------------------+
| 0 (flags) |     4 (length)       |
+----------------------------------|
|            32-bit ID             |
+----------------------------------+

*/
type PingFrame struct {
	CtrlFrameHead

	PingId uint32
}

/*

GOAWAY


+----------------------------------+
|1|       2          |       7     |
+----------------------------------+
| 0 (flags) |     4 (length)       |
+----------------------------------|
|X|  Last-good-stream-ID (31 bits) |
+----------------------------------+

*/
type GoawayFrame struct {
	CtrlFrameHead

	LastGoodId uint32
}

/*

HEADERS

+----------------------------------+
|C|     2           |      8       |
+----------------------------------+
| Flags (8)  |  Length (24 bits)   |
+----------------------------------+
|X|          Stream-ID (31bits)    |
+----------------------------------+
|  Unused (16 bits) |              |
|--------------------              |
| Name/value header block          |
+----------------------------------+

*/
type HeadersFrame struct {
	CtrlFrameHead

	StreamId uint32
	Unused   uint16

	Header map[string]string
}

// HeaderDictionary is the dictionary sent to the zlib compressor/decompressor.
// Even though the specification states there is no null byte at the end, Chrome sends it.
const HeaderDict = "optionsgetheadpostputdeletetraceacceptaccept-charsetaccept-encodingaccept-" +
	"languageauthorizationexpectfromhostif-modified-sinceif-matchif-none-matchi" +
	"f-rangeif-unmodifiedsincemax-forwardsproxy-authorizationrangerefererteuser" +
	"-agent10010120020120220320420520630030130230330430530630740040140240340440" +
	"5406407408409410411412413414415416417500501502503504505accept-rangesageeta" +
	"glocationproxy-authenticatepublicretry-afterservervarywarningwww-authentic" +
	"ateallowcontent-basecontent-encodingcache-controlconnectiondatetrailertran" +
	"sfer-encodingupgradeviawarningcontent-languagecontent-lengthcontent-locati" +
	"oncontent-md5content-rangecontent-typeetagexpireslast-modifiedset-cookieMo" +
	"ndayTuesdayWednesdayThursdayFridaySaturdaySundayJanFebMarAprMayJunJulAugSe" +
	"pOctNovDecchunkedtext/htmlimage/pngimage/jpgimage/gifapplication/xmlapplic" +
	"ation/xhtmltext/plainpublicmax-agecharset=iso-8859-1utf-8gzipdeflateHTTP/1" +
	".1statusversionurl\x00"
