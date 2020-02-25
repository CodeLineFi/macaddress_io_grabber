package models

import (
	"encoding/json"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"math/big"
	"strings"
)

var rangesSearchScript *redis.Script

func init() {
	rangesSearchScript = redis.NewScript(0, `
	local i = 0
	local key = ARGV[1]
	local j = redis.call("HLEN", key) - 1
	local n = ARGV[2]
	local mac_address_length = tonumber(string.sub(n,1,2),16)

	while (i < j) do

		local h = math.ceil((i + j) / 2)

		local c = redis.call("HGET", key, h)
		c = string.sub(c, 1, 2 + mac_address_length*2)

		if c <= n then
		i = h
		else
		j = h - 1
		end
	end

	return redis.call("HGET", key, i)
	`)
}

type Ranger interface {
	LeftBorder() MAC
	RightBorder() MAC
	GetRangeID() uint64
	SaveToRedis(client redis.Conn, redisKey string) error
	Length() *big.Int
}

type RangeJSON struct {
	LeftBorder  string `json:"leftBorder"`
	RightBorder string `json:"rightBorder"`
}

type RangeRedis struct {
	RangeID uint64   `json:"vmRangeID"`
	LBorder MAC      `json:"lBorder"`
	RBorder MAC      `json:"rBorder"`
	length  *big.Int `json:"-"`
}

func (model *RangeRedis) Length() *big.Int {
	if model.length == nil {
		left, _ := new(big.Int).SetString(model.LBorder.String(), 16)
		right, _ := new(big.Int).SetString(model.RBorder.String(), 16)

		model.length = right.Sub(right, left)
	}

	return model.length
}

func (model *RangeRedis) LeftBorder() MAC {
	return model.LBorder
}

func (model *RangeRedis) RightBorder() MAC {
	return model.RBorder
}

type VMRangeJSON struct {
	RangeJSON
	VirtualMachineName string `json:"virtualMachineName"`
	Reference          string `json:"reference"`
}

type VMRangeRedis struct {
	RangeRedis
	VirtualMachineName string   `json:"vmName"`
	length             *big.Int `json:"-"`
}

func (rng VMRangeRedis) GetRangeID() uint64 {
	return rng.RangeID
}

func (rng *VMRangeRedis) SaveToRedis(client redis.Conn, redisKey string) error {
	return saveRangeToRedis(client, rng, redisKey)
}

type ApplicationRangeJSON struct {
	RangeJSON
	Application string `json:"application"`
	Notes       string `json:"notes"`
	Reference   string `json:"reference"`
}

type ApplicationRangeRedis struct {
	RangeRedis
	Application string   `json:"application"`
	Notes       string   `json:"notes"`
	length      *big.Int `json:"-"`
}

func (rng *ApplicationRangeRedis) GetRangeID() uint64 {
	return rng.RangeID
}

func (rng *ApplicationRangeRedis) SaveToRedis(client redis.Conn, redisKey string) error {
	return saveRangeToRedis(client, rng, redisKey)
}

func GetNilRangeHashTag() string {
	return "nil"
}

func FindVMRanges(conn redis.Conn, search *MAC) ([]*VMRangeRedis, error) {
	ranges, err := findRanges(
		conn,
		search,
		RedisVmRangeSearchTable,
		RedisVmRangeHash,
	)
	if err != nil {
		return nil, err
	}

	vmRanges := make([]*VMRangeRedis, 0, len(ranges))

	for _, rng := range ranges {
		_range := &VMRangeRedis{}
		err = json.Unmarshal([]byte(rng), _range)

		if err != nil {
			return vmRanges, err
		}
		vmRanges = append(vmRanges, _range)
	}

	return vmRanges, nil
}

func FindApplicationsRanges(conn redis.Conn, search *MAC) ([]*ApplicationRangeRedis, error) {
	ranges, err := findRanges(
		conn,
		search,
		RedisAppRangeSearchTable,
		RedisAppRangeHash,
	)
	if err != nil {
		return nil, err
	}

	appRanges := make([]*ApplicationRangeRedis, 0, len(ranges))

	for _, rng := range ranges {
		_range := &ApplicationRangeRedis{}
		err = json.Unmarshal([]byte(rng), _range)

		if err != nil {
			return appRanges, err
		}
		appRanges = append(appRanges, _range)
	}

	return appRanges, nil
}

func findRanges(
	conn redis.Conn,
	search *MAC,
	searchTable string,
	rangesHash string,
) ([]string, error) {
	if search == nil {
		return []string{}, nil
	}

	var slice []byte
	var err error

	switch search.Length() {
	case 6, 7, 9:
		mac := search.Resize(12)
		slice, err = redis.Bytes(rangesSearchScript.Do(conn, searchTable, ToIndex(mac)))
	case 12, 16:
		slice, err = redis.Bytes(rangesSearchScript.Do(conn, searchTable, ToIndex(*search)))
	}

	if err != nil {
		return nil, err
	}

	if len(slice) == 0 {
		return []string{}, nil
	}

	hashes := strings.Split(string(slice), ";")[1:]

	if 0 == len(hashes) ||
		strings.Join(hashes, ";") == GetNilRangeHashTag() {
		return nil, nil
	}

	ranges := make([]string, 0, len(hashes))

	for _, hash := range hashes {
		rng, err := redis.String(conn.Do("HGET", rangesHash, hash))

		if err != nil {
			return ranges, err
		}

		ranges = append(ranges, rng)
	}

	return ranges, nil
}

func saveRangeToRedis(client redis.Conn, rng Ranger, redisKey string) error {

	object, err := json.Marshal(rng)
	if err != nil {
		return err
	}

	if _, err := client.Do(
		"HMSET",
		redisKey,
		rng.GetRangeID(),
		object,
	); err != nil {
		return err
	}

	return nil
}

func ToIndex(mac MAC) string {
	mac = mac.Ceil()
	return fmt.Sprintf("%02x%s", mac.Length()/2, mac.String())
}
