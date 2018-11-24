package snowflake

/*
 * Desc: snowflake算法（分布式唯一整型id生成器）
 * Author: Sakishum
 * Date: 2018年 11月 22日
 * URL: https://github.com/twitter-archive/snowflake
 */

import (
 	"encoding/base64"
 	"encoding/binary"
	"strconv"
	"errors"
	"sync"
	"time"
	"fmt"
)

const (
	// Epoch is set to the twitter snowflake epoch of Nov 04 2010 01:42:54 UTC
	// You may customize this to set a different epoch for your application.
	twepoch        = int64(1542944160000) // 默认起始的时间戳 1542944160000 。计算时，减去这个值
	DistrictIdBits = uint(5)              // 区域 所占用位置
	NodeIdBits     = uint(9)              // 节点 所占位置, 2^9 = 512
	sequenceBits   = uint(10)             // 自增 ID 所占用位置, 每秒上限 1000 * (2^10) = 102.4w

	/*
	 * snowflake-64bit :
	 * 1 符号位	|  39 时间戳										| 5 区域	| 9 节点			| 10 （毫秒内）自增ID
	 * 0		|  0000000 00000000 00000000 00000000 00000000	| 00000	| 000000 000	| 000000 0000
	 *
	 * 1. 39 位时间截(毫秒级)，注意这是时间截的差值（当前时间截 - 开始时间截)。可以使用约: (1L << 39) / (1000L * 60 * 60 * 24 * 365)
	 * 2. 9  位数据机器位，可以部署在 512 个节点
	 * 3. 10 位序列，毫秒内的计数，同一机器，同一时间截并发 1024 个序号
	 */
	maxNodeId     = -1 ^ (-1 << NodeIdBits)     // 节点 ID 最大范围
	maxDistrictId = -1 ^ (-1 << DistrictIdBits) // 最大区域范围

	nodeIdShift        	= sequenceBits 	// 左移次数
	districtIdShift		= sequenceBits + NodeIdBits
	timestampLeftShift	= sequenceBits + NodeIdBits + DistrictIdBits
	sequenceMask       	= -1 ^ (-1 << sequenceBits)
	nodeIdMask			= maxNodeId << sequenceBits
	districtMask		= maxDistrictId << districtIdShift
	maxNextIdsNum		= 100 			// 单次获取ID的最大数量
)

type IdWorker struct {
	sync.Mutex
	sequence      int64 // 序号
	lastTimestamp int64 // 最后时间戳
	nodeId        int64 // 节点 ID
	twepoch       int64 // 起始时间戳
	districtId    int64 // 区域 ID
}

type ID int64

// NewIdWorker new a snowflake id generator object.
func NewIdWorker(NodeId int64) (*IdWorker, error) {
	var districtId int64
	districtId = 1 // 暂时默认给1 ，方便以后扩展
	if NodeId > maxNodeId || NodeId < 0 {
		//fmt.Sprintf("NodeId Id can't be greater than %d or less than 0", maxNodeId)
		return nil, errors.New(fmt.Sprintf("workerid must be between 0 and %d", maxNodeId))
	}
	if districtId > maxDistrictId || districtId < 0 {
		//fmt.Sprintf("District Id can't be greater than %d or less than 0", maxDistrictId)
		return nil, errors.New(fmt.Sprintf("district must be between 0 and %d", maxDistrictId))
	}

	//fmt.Printf("worker starting. timestamp left shift %d, District id bits %d, worker id bits %d, sequence bits %d, workerid %d\n", timestampLeftShift, DistrictIdBits, NodeIdBits, sequenceBits, NodeId)
	return &IdWorker{
		nodeId:        NodeId,
		districtId:    districtId,
		lastTimestamp: -1,
		sequence:      0,
		twepoch:       twepoch,
	}, nil
}

// timeGen generate a unix millisecond.
func timeGen() int64 {
	// 当前纳秒 / 1e6 = 当前毫秒
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// tilNextMillis spin wait till next millisecond.
func tilNextMillis(lastTimestamp int64) int64 {
	timestamp := timeGen()
	for timestamp <= lastTimestamp {
		timestamp = timeGen()
	}
	return timestamp
}

// NextId get a snowflake id.
func (id *IdWorker) NextId() (ID, error) {
	id.Lock()
	defer id.Unlock()
	return id.nextid()
}

// NextIds get snowflake ids.
func (id *IdWorker) NextIds(num int) ([]ID, error) {
	if num > maxNextIdsNum || num < 0 {
		//fmt.Printf("NextIds num can't be greater than %d or less than 0\n", maxNextIdsNum)
		return nil, errors.New(fmt.Sprintf("NextIds num: %d error", num))
	}
	ids := make([]ID, num)
	id.Lock()
	defer id.Unlock()
	for i := 0; i < num; i++ {
		ids[i], _ = id.nextid()
	}
	return ids, nil
}

func (id *IdWorker) nextid() (ID, error) {
	timestamp := timeGen()
	if timestamp < id.lastTimestamp {
		return 0, errors.New(fmt.Sprintf("Clock moved backwards.  Refusing to generate id for %d milliseconds", id.lastTimestamp-timestamp))
	}
	if id.lastTimestamp == timestamp {
		id.sequence = (id.sequence + 1) & sequenceMask
		if id.sequence == 0 {
			timestamp = tilNextMillis(id.lastTimestamp)
		}
	} else {
		id.sequence = 0
	}
	id.lastTimestamp = timestamp
	return ID(((timestamp - id.twepoch) << timestampLeftShift) | (id.districtId << districtIdShift) | (id.nodeId << nodeIdShift) | id.sequence), nil
}

func (f ID) Time() int64 {
	return ((int64(f) >> timestampLeftShift) + twepoch) / 1e3
}

func (f ID) NodeId() int64 {
	return int64(f) & nodeIdMask >> nodeIdShift
}

func (f ID) DistrictId() int64 {
	return int64(f) & districtMask >> districtIdShift
}

func (f ID) Int64() int64 {
	return int64(f)
}

func (f ID) String() string {
	return strconv.FormatInt(int64(f), 10)
}

func (f ID) Bytes() []byte {
	return []byte(f.String())
}

func (f ID) IntBytes() [8]byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(f))
	return b
}

func (f ID) Base64() string {
	return base64.StdEncoding.EncodeToString(f.Bytes())
}

