package collector

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/maruel/natural"
)

func debugPrint(dutum []MetricsDutum) {
	var dutumStr []string
	for i := range dutum {
		dutumStr = append(dutumStr, dutum[i].String())
	}
	sort.Sort(natural.StringSlice(dutumStr))
	// debug print.
	fmt.Print("\033[H\033[2J")
	fmt.Println(time.Now().Format(time.ANSIC))
	fmt.Println(strings.Join(dutumStr, "\n"))
}
