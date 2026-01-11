package changeset

import (
	"fmt"
	"math/rand"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

// adjectives is a list of positive, friendly adjectives (150 words)
var adjectives = []string{
	"amazing", "awesome", "bold", "brave", "bright", "brilliant", "calm", "capable",
	"charming", "cheerful", "clever", "confident", "cool", "courageous", "creative",
	"dazzling", "delightful", "determined", "dynamic", "eager", "elegant", "energetic",
	"enthusiastic", "excellent", "fabulous", "fancy", "fantastic", "fearless", "festive",
	"fierce", "fine", "friendly", "generous", "gentle", "gifted", "glorious", "golden",
	"graceful", "grand", "great", "happy", "harmonious", "honest", "hopeful", "incredible",
	"inspired", "jolly", "joyful", "kind", "lively", "lovely", "loyal", "lucky",
	"magnificent", "majestic", "marvelous", "merry", "mighty", "neat", "nice", "noble",
	"optimistic", "peaceful", "perfect", "pleasant", "polite", "powerful", "proud", "quick",
	"quiet", "radiant", "reliable", "remarkable", "resilient", "respectful", "shiny",
	"sincere", "skilled", "smart", "smooth", "sparkling", "splendid", "stellar", "strong",
	"stunning", "superb", "sweet", "talented", "tender", "terrific", "thoughtful", "tidy",
	"tranquil", "tremendous", "trusted", "unique", "upbeat", "valiant", "vibrant",
	"victorious", "vigorous", "warm", "willing", "wise", "witty", "wonderful", "worthy",
	"zealous", "zippy",
}

// animals is a list of friendly animals (200+ words)
var animals = []string{
	// Mammals
	"aardvark", "alpaca", "antelope", "armadillo", "badger", "bat", "bear", "beaver",
	"bison", "bobcat", "buffalo", "camel", "capybara", "cat", "cheetah", "chipmunk",
	"cougar", "coyote", "deer", "dingo", "dog", "dolphin", "donkey", "elephant", "elk",
	"ferret", "fox", "gazelle", "gerbil", "giraffe", "goat", "groundhog", "hamster",
	"hare", "hedgehog", "hippo", "horse", "hyena", "impala", "jackal", "jaguar",
	"kangaroo", "koala", "lemur", "leopard", "lion", "llama", "lynx", "meerkat", "mink",
	"mole", "mongoose", "monkey", "moose", "mouse", "mule", "ocelot", "opossum",
	"orangutan", "orca", "otter", "ox", "panda", "panther", "pig", "platypus", "pony",
	"porcupine", "possum", "puma", "rabbit", "raccoon", "ram", "rat", "reindeer", "rhino",
	"seal", "sheep", "shrew", "skunk", "sloth", "squirrel", "tapir", "tiger", "vole",
	"walrus", "weasel", "whale", "wildcat", "wolf", "wolverine", "wombat", "yak", "zebra",
	// Birds
	"albatross", "bluebird", "cardinal", "chickadee", "cockatoo", "condor", "crane",
	"crow", "cuckoo", "dove", "duck", "eagle", "egret", "falcon", "finch", "flamingo",
	"goose", "grouse", "gull", "hawk", "heron", "hummingbird", "ibis", "jay", "kestrel",
	"kingfisher", "kite", "lark", "loon", "magpie", "mallard", "martin", "mockingbird",
	"nightingale", "oriole", "osprey", "ostrich", "owl", "parakeet", "parrot", "peacock",
	"pelican", "penguin", "pheasant", "pigeon", "plover", "puffin", "quail", "raven",
	"robin", "sandpiper", "sparrow", "starling", "stork", "swallow", "swan", "swift",
	"tanager", "tern", "thrush", "toucan", "turkey", "vulture", "warbler", "woodpecker",
	"wren",
	// Reptiles & Amphibians
	"chameleon", "cobra", "crocodile", "frog", "gecko", "iguana", "lizard", "newt",
	"python", "salamander", "snake", "toad", "tortoise", "turtle",
	// Fish & Aquatic
	"anchovy", "bass", "carp", "catfish", "clownfish", "cod", "eel", "goldfish", "guppy",
	"halibut", "herring", "jellyfish", "mackerel", "marlin", "minnow", "octopus", "perch",
	"pike", "salmon", "sardine", "seahorse", "shark", "snapper", "squid", "starfish",
	"swordfish", "trout", "tuna", "walleye",
	// Insects
	"ant", "bee", "beetle", "butterfly", "cricket", "dragonfly", "firefly", "grasshopper",
	"ladybug", "mantis", "moth",
}

var rng *rand.Rand

func init() {
	// Initialize random number generator with current time
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// generateHumanFriendlyID generates a human-friendly ID like "dazzling_mouse_V1StGXR8"
func generateHumanFriendlyID() (string, error) {
	// Pick random adjective and animal
	adjective := adjectives[rng.Intn(len(adjectives))]
	animal := animals[rng.Intn(len(animals))]

	// Generate short nanoid (8 characters instead of default 21)
	nanoID, err := gonanoid.Generate("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", 8)
	if err != nil {
		return "", fmt.Errorf("failed to generate nanoid: %w", err)
	}

	// Combine: adjective_animal_nanoid
	return fmt.Sprintf("%s_%s_%s", adjective, animal, nanoID), nil
}
