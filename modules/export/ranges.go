package export

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"log"
	"macaddress_io_grabber/models"
	"macaddress_io_grabber/utils/database"
	"sort"
	"strconv"
	"strings"
)

func appRanges(client redis.Conn) error {
	var appRanges []database.ApplicationRange

	database.Instance.Find(&appRanges)

	leftBordersUnique := map[string]struct{}{}

	appRangesRedis := make([]*models.ApplicationRangeRedis, 0, len(appRanges))

	for _, appRange := range appRanges {

		leftBorder, err := models.NewMac(appRange.LeftBorder)

		if err != nil {
			return err
		}

		rightBorder, err := models.NewMac(appRange.RightBorder)

		if err != nil {
			return err
		}

		if leftBorder.Length() != rightBorder.Length() || leftBorder.String() > rightBorder.String() {
			log.Println("cannot process app range: " + leftBorder.String() + " - " + rightBorder.String())
			continue
		}

		redisModel := models.ApplicationRangeRedis{
			RangeRedis: models.RangeRedis{
				RangeID: appRange.ID,
				LBorder: leftBorder,
				RBorder: rightBorder,
			},
			Application: appRange.Application,
		}
		appRangesRedis = append(appRangesRedis, &redisModel)

		if err = redisModel.SaveToRedis(client, models.RedisAppRangeHash); err != nil {
			return err
		}

		collectLeftBorders(
			leftBorder,
			rightBorder,
			leftBordersUnique,
		)
	}

	leftBorders := make([]string, 0, len(leftBordersUnique))

	for key := range leftBordersUnique {
		leftBorders = append(leftBorders, key)
	}

	sort.Slice(leftBorders, func(i, j int) bool {
		return leftBorders[i] < leftBorders[j]
	})

	rangesBordersMap := make([][]models.Ranger, len(leftBorders))

	for _, rng := range appRangesRedis {
		if err := calculateRangesSearchHash(
			leftBorders,
			rangesBordersMap,
			rng,
		); err != nil {
			return err
		}
	}

	return saveSearchHashToRedis(client, leftBorders, rangesBordersMap, models.RedisAppRangeSearchTable)
}

func vmRanges(client redis.Conn) error {
	var vmRanges []database.VMRange

	database.Instance.Find(&vmRanges)

	leftBordersUnique := map[string]struct{}{}
	vmRangesRedis := make([]*models.VMRangeRedis, 0, len(vmRanges))

	for _, vmRange := range vmRanges {

		leftBorder, err := models.NewMac(vmRange.LeftBorder)
		if err != nil {
			return err
		}

		rightBorder, err := models.NewMac(vmRange.RightBorder)
		if err != nil {
			return err
		}

		if leftBorder.Length() != rightBorder.Length() || leftBorder.String() > rightBorder.String() {
			log.Println("cannot process vm range: " + leftBorder.String() + " - " + rightBorder.String())
			continue
		}

		redisModel := models.VMRangeRedis{
			RangeRedis: models.RangeRedis{
				RangeID: vmRange.ID,
				LBorder: leftBorder,
				RBorder: rightBorder,
			},
			VirtualMachineName: vmRange.VirtualMachineName,
		}

		vmRangesRedis = append(vmRangesRedis, &redisModel)
		if err = redisModel.SaveToRedis(client, models.RedisVmRangeHash); err != nil {
			return err
		}

		collectLeftBorders(
			leftBorder,
			rightBorder,
			leftBordersUnique,
		)

	}

	leftBorders := make([]string, 0, len(leftBordersUnique))

	for key := range leftBordersUnique {
		leftBorders = append(leftBorders, key)
	}

	sort.Slice(leftBorders, func(i, j int) bool {
		return leftBorders[i] < leftBorders[j]
	})

	rangesBordersMap := make([][]models.Ranger, len(leftBorders))

	for _, rng := range vmRangesRedis {
		if err := calculateRangesSearchHash(
			leftBorders,
			rangesBordersMap,
			rng,
		); err != nil {
			return err
		}
	}

	return saveSearchHashToRedis(client, leftBorders, rangesBordersMap, models.RedisVmRangeSearchTable)
}

func calculateRangesSearchHash(
	leftBorders []string,
	rangesBordersMap [][]models.Ranger,
	rng models.Ranger,
) error {

	leftBorder := models.ToIndex(rng.LeftBorder())
	rightBorder := models.ToIndex(rng.RightBorder())

	l := sort.Search(len(leftBorders), func(i int) bool {
		return leftBorders[i] >= leftBorder
	})

	r := sort.Search(len(leftBorders), func(i int) bool {
		return leftBorders[i] >= rightBorder
	})

	for i := l; i < r; i++ {
		rangesBordersMap[i] = append(rangesBordersMap[i], rng)
	}

	return nil
}

func sortRangesBorders(rangesBordersMap [][]models.Ranger) {
	for _, rangesBorders := range rangesBordersMap {
		sort.Slice(rangesBorders, func(i, j int) bool {
			switch rangesBorders[i].Length().Cmp(rangesBorders[j].Length()) {
			case -1:
				return true
			case 0:
				return rangesBorders[i].GetRangeID() < rangesBorders[j].GetRangeID()
			case 1:
				return false
			default:
				panic("not expected value")
			}
		})
	}
}

func saveSearchHashToRedis(
	client redis.Conn,
	leftBorders []string,
	rangesBordersMap [][]models.Ranger,
	redisTag string,
) error {

	sortRangesBorders(rangesBordersMap)

	var hash = 0

	for i, leftBorder := range leftBorders {
		var target string

		if 0 == len(rangesBordersMap[i]) {
			target = models.GetNilRangeHashTag()
		} else {
			targets := make([]string, 0, len(rangesBordersMap[i]))
			for _, rangeObject := range rangesBordersMap[i] {
				targets = append(targets, strconv.FormatUint(rangeObject.GetRangeID(), 10))
			}

			target = strings.Join(targets, ";")
		}

		if _, err := client.Do(
			hmset,
			redisTag,
			hash,
			fmt.Sprintf("%s;%s", leftBorder, target),
		); err != nil {
			return err
		}

		hash++
	}

	return nil
}

func collectLeftBorders(leftBorder models.MAC, rightBorder models.MAC, collection map[string]struct{}) {
	collection[models.ToIndex(leftBorder)] = struct{}{}
	collection[models.ToIndex(rightBorder.Incr())] = struct{}{}
}
