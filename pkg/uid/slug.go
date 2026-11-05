package uid

import (
	"fmt"
	"math/rand/v2"
)

var (
	slugAdjectives = []string{
		"amber", "bold", "brave", "bright", "calm", "clever", "cosmic", "crisp",
		"daring", "eager", "fancy", "gentle", "golden", "happy", "humble", "icy",
		"jolly", "keen", "lively", "lucky", "mellow", "misty", "noble", "polite",
		"proud", "quick", "quiet", "rapid", "rustic", "shiny", "silent", "silver",
		"sleek", "smooth", "snowy", "solid", "steady", "sunny", "swift", "tidy",
		"vivid", "warm", "wild", "wise", "witty", "young", "zesty", "zippy",
	}
	slugNouns = []string{
		"anchor", "badger", "beacon", "breeze", "canyon", "cedar", "comet", "coral",
		"falcon", "fern", "forest", "fox", "galaxy", "glacier", "harbor", "hawk",
		"heron", "island", "lagoon", "lantern", "maple", "meadow", "meteor", "otter",
		"owl", "panda", "pebble", "pine", "planet", "prairie", "raven", "reef",
		"river", "rocket", "sparrow", "spruce", "summit", "thunder", "tiger", "trail",
		"valley", "violet", "walrus", "willow", "wolf", "yak", "zebra", "zephyr",
	}
)

// Slug generates a readable random slug such as "brave-otter-4821".
//
// The result is lowercase letters, digits, and single hyphens, starts with a
// letter, and fits both resource slugs and DNS-1035 labels. The numeric suffix
// keeps collisions unlikely across repeated test runs against shared state.
//
// Uses math/rand/v2 which is NOT cryptographically secure.
func Slug() string {
	return fmt.Sprintf("%s-%s-%04d",
		slugAdjectives[rand.IntN(len(slugAdjectives))],
		slugNouns[rand.IntN(len(slugNouns))],
		rand.IntN(10000),
	)
}
