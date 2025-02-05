package lsp

var EmojiMapper = map[string]struct {
	Emoji string
	Desc  string
}{
	"happy": {"😀", "Feeling like a million bucks!"},
	"sad":   {"😢", "When your pizza delivery gets delayed..."},
	"angry": {"😠", "When your favorite TV show gets canceled."},
	"confused": {
		"😕",
		"Trying to understand why the chicken crossed the road.",
	},
	"excited":   {"😆", "That moment when you find an extra fry in the bag."},
	"love":      {"😍", "When you see the last donut at the office."},
	"laughing":  {"😂", "That one joke that gets funnier every time."},
	"crying":    {"😭", "When you drop your ice cream on the floor."},
	"sleepy":    {"😴", "That post-lunch nap you're planning for."},
	"surprised": {"😮", "When you realize it’s already Friday."},
	"sick": {
		"🤒",
		"Feeling like you’re about to join the zombie apocalypse.",
	},
	"cool": {"😎", "When you walk into the room and everyone notices."},
	"nerd": {"🤓", "When you can name every character from Star Wars."},
	"worried": {
		"😟",
		"When you can’t find your phone... and it’s in your hand.",
	},
	"scared": {"😨", "That moment when you hear a knock in the dark."},
	"silly": {
		"🤪",
		"When you try to do a serious face, but can’t stop giggling.",
	},
	"shocked":    {"😱", "When the Wi-Fi goes down during a movie."},
	"sunglasses": {"😎", "When you’re just too cool for school."},
	"tongue":     {"😛", "When you make a silly face for no reason."},
	"thinking": {
		"🤔",
		"Trying to figure out why socks always disappear in the laundry.",
	},
}
