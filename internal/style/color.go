package style

import "strings"

// Color is a non-premultiplied sRGB color with alpha, 8 bits per channel.
type Color struct{ R, G, B, A uint8 }

// parseColor parses a CSS color: #rgb, #rgba, #rrggbb, #rrggbbaa,
// rgb()/rgba() (comma or space syntax, numbers or percentages, optional
// alpha), a named color, `transparent`, or `currentcolor` (reported with
// current=true). s must be lower-case.
func parseColor(s string) (c Color, current, ok bool) {
	switch {
	case s == "currentcolor":
		return Color{}, true, true
	case s == "transparent":
		return Color{}, false, true
	case strings.HasPrefix(s, "#"):
		c, ok = parseHex(s[1:])
		return c, false, ok
	case strings.HasPrefix(s, "rgb(") || strings.HasPrefix(s, "rgba("):
		if !strings.HasSuffix(s, ")") {
			return Color{}, false, false
		}
		c, ok = parseRGB(s[strings.IndexByte(s, '(')+1 : len(s)-1])
		return c, false, ok
	}
	c, ok = namedColors[s]
	return c, false, ok
}

func hexVal(b byte) (uint8, bool) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', true
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10, true
	}
	return 0, false
}

func parseHex(h string) (Color, bool) {
	var d [8]uint8
	if len(h) != 3 && len(h) != 4 && len(h) != 6 && len(h) != 8 {
		return Color{}, false
	}
	for i := 0; i < len(h); i++ {
		v, ok := hexVal(h[i])
		if !ok {
			return Color{}, false
		}
		d[i] = v
	}
	switch len(h) {
	case 3, 4:
		c := Color{d[0] * 17, d[1] * 17, d[2] * 17, 255}
		if len(h) == 4 {
			c.A = d[3] * 17
		}
		return c, true
	default:
		c := Color{d[0]<<4 | d[1], d[2]<<4 | d[3], d[4]<<4 | d[5], 255}
		if len(h) == 8 {
			c.A = d[6]<<4 | d[7]
		}
		return c, true
	}
}

// parseRGB parses the inside of rgb()/rgba(): "r, g, b[, a]" or
// "r g b[ / a]".
func parseRGB(in string) (Color, bool) {
	f := strings.Fields(strings.NewReplacer(",", " ", "/", " / ").Replace(in))
	var parts []string
	for _, x := range f {
		if x != "/" {
			parts = append(parts, x)
		}
	}
	if len(parts) != 3 && len(parts) != 4 {
		return Color{}, false
	}
	var ch [4]uint8
	ch[3] = 255
	for i, p := range parts {
		var v int64
		var ok bool
		switch {
		case strings.HasSuffix(p, "%"):
			// percent (scale 1000) -> 0..255, rounded
			var pc int64
			pc, ok = parseFixed(p[:len(p)-1], 1000)
			v = roundDiv(pc*255, 100*1000)
		case i == 3:
			// alpha number 0..1 (scale 1e6) -> 0..255
			var a int64
			a, ok = parseFixed(p, 1_000_000)
			v = roundDiv(a*255, 1_000_000)
		default:
			var n int64
			n, ok = parseFixed(p, 1_000_000)
			v = roundDiv(n, 1_000_000)
		}
		if !ok {
			return Color{}, false
		}
		ch[i] = uint8(min(max(v, 0), 255))
	}
	return Color{ch[0], ch[1], ch[2], ch[3]}, true
}

// namedColors are the CSS Color Module Level 4 named colors.
var namedColors = map[string]Color{
	"aliceblue": {240, 248, 255, 255}, "antiquewhite": {250, 235, 215, 255}, "aqua": {0, 255, 255, 255},
	"aquamarine": {127, 255, 212, 255}, "azure": {240, 255, 255, 255}, "beige": {245, 245, 220, 255},
	"bisque": {255, 228, 196, 255}, "black": {0, 0, 0, 255}, "blanchedalmond": {255, 235, 205, 255},
	"blue": {0, 0, 255, 255}, "blueviolet": {138, 43, 226, 255}, "brown": {165, 42, 42, 255},
	"burlywood": {222, 184, 135, 255}, "cadetblue": {95, 158, 160, 255}, "chartreuse": {127, 255, 0, 255},
	"chocolate": {210, 105, 30, 255}, "coral": {255, 127, 80, 255}, "cornflowerblue": {100, 149, 237, 255},
	"cornsilk": {255, 248, 220, 255}, "crimson": {220, 20, 60, 255}, "cyan": {0, 255, 255, 255},
	"darkblue": {0, 0, 139, 255}, "darkcyan": {0, 139, 139, 255}, "darkgoldenrod": {184, 134, 11, 255},
	"darkgray": {169, 169, 169, 255}, "darkgreen": {0, 100, 0, 255}, "darkgrey": {169, 169, 169, 255},
	"darkkhaki": {189, 183, 107, 255}, "darkmagenta": {139, 0, 139, 255}, "darkolivegreen": {85, 107, 47, 255},
	"darkorange": {255, 140, 0, 255}, "darkorchid": {153, 50, 204, 255}, "darkred": {139, 0, 0, 255},
	"darksalmon": {233, 150, 122, 255}, "darkseagreen": {143, 188, 143, 255}, "darkslateblue": {72, 61, 139, 255},
	"darkslategray": {47, 79, 79, 255}, "darkslategrey": {47, 79, 79, 255}, "darkturquoise": {0, 206, 209, 255},
	"darkviolet": {148, 0, 211, 255}, "deeppink": {255, 20, 147, 255}, "deepskyblue": {0, 191, 255, 255},
	"dimgray": {105, 105, 105, 255}, "dimgrey": {105, 105, 105, 255}, "dodgerblue": {30, 144, 255, 255},
	"firebrick": {178, 34, 34, 255}, "floralwhite": {255, 250, 240, 255}, "forestgreen": {34, 139, 34, 255},
	"fuchsia": {255, 0, 255, 255}, "gainsboro": {220, 220, 220, 255}, "ghostwhite": {248, 248, 255, 255},
	"gold": {255, 215, 0, 255}, "goldenrod": {218, 165, 32, 255}, "gray": {128, 128, 128, 255},
	"green": {0, 128, 0, 255}, "greenyellow": {173, 255, 47, 255}, "grey": {128, 128, 128, 255},
	"honeydew": {240, 255, 240, 255}, "hotpink": {255, 105, 180, 255}, "indianred": {205, 92, 92, 255},
	"indigo": {75, 0, 130, 255}, "ivory": {255, 255, 240, 255}, "khaki": {240, 230, 140, 255},
	"lavender": {230, 230, 250, 255}, "lavenderblush": {255, 240, 245, 255}, "lawngreen": {124, 252, 0, 255},
	"lemonchiffon": {255, 250, 205, 255}, "lightblue": {173, 216, 230, 255}, "lightcoral": {240, 128, 128, 255},
	"lightcyan": {224, 255, 255, 255}, "lightgoldenrodyellow": {250, 250, 210, 255}, "lightgray": {211, 211, 211, 255},
	"lightgreen": {144, 238, 144, 255}, "lightgrey": {211, 211, 211, 255}, "lightpink": {255, 182, 193, 255},
	"lightsalmon": {255, 160, 122, 255}, "lightseagreen": {32, 178, 170, 255}, "lightskyblue": {135, 206, 250, 255},
	"lightslategray": {119, 136, 153, 255}, "lightslategrey": {119, 136, 153, 255}, "lightsteelblue": {176, 196, 222, 255},
	"lightyellow": {255, 255, 224, 255}, "lime": {0, 255, 0, 255}, "limegreen": {50, 205, 50, 255},
	"linen": {250, 240, 230, 255}, "magenta": {255, 0, 255, 255}, "maroon": {128, 0, 0, 255},
	"mediumaquamarine": {102, 205, 170, 255}, "mediumblue": {0, 0, 205, 255}, "mediumorchid": {186, 85, 211, 255},
	"mediumpurple": {147, 112, 219, 255}, "mediumseagreen": {60, 179, 113, 255}, "mediumslateblue": {123, 104, 238, 255},
	"mediumspringgreen": {0, 250, 154, 255}, "mediumturquoise": {72, 209, 204, 255}, "mediumvioletred": {199, 21, 133, 255},
	"midnightblue": {25, 25, 112, 255}, "mintcream": {245, 255, 250, 255}, "mistyrose": {255, 228, 225, 255},
	"moccasin": {255, 228, 181, 255}, "navajowhite": {255, 222, 173, 255}, "navy": {0, 0, 128, 255},
	"oldlace": {253, 245, 230, 255}, "olive": {128, 128, 0, 255}, "olivedrab": {107, 142, 35, 255},
	"orange": {255, 165, 0, 255}, "orangered": {255, 69, 0, 255}, "orchid": {218, 112, 214, 255},
	"palegoldenrod": {238, 232, 170, 255}, "palegreen": {152, 251, 152, 255}, "paleturquoise": {175, 238, 238, 255},
	"palevioletred": {219, 112, 147, 255}, "papayawhip": {255, 239, 213, 255}, "peachpuff": {255, 218, 185, 255},
	"peru": {205, 133, 63, 255}, "pink": {255, 192, 203, 255}, "plum": {221, 160, 221, 255},
	"powderblue": {176, 224, 230, 255}, "purple": {128, 0, 128, 255}, "rebeccapurple": {102, 51, 153, 255},
	"red": {255, 0, 0, 255}, "rosybrown": {188, 143, 143, 255}, "royalblue": {65, 105, 225, 255},
	"saddlebrown": {139, 69, 19, 255}, "salmon": {250, 128, 114, 255}, "sandybrown": {244, 164, 96, 255},
	"seagreen": {46, 139, 87, 255}, "seashell": {255, 245, 238, 255}, "sienna": {160, 82, 45, 255},
	"silver": {192, 192, 192, 255}, "skyblue": {135, 206, 235, 255}, "slateblue": {106, 90, 205, 255},
	"slategray": {112, 128, 144, 255}, "slategrey": {112, 128, 144, 255}, "snow": {255, 250, 250, 255},
	"springgreen": {0, 255, 127, 255}, "steelblue": {70, 130, 180, 255}, "tan": {210, 180, 140, 255},
	"teal": {0, 128, 128, 255}, "thistle": {216, 191, 216, 255}, "tomato": {255, 99, 71, 255},
	"turquoise": {64, 224, 208, 255}, "violet": {238, 130, 238, 255}, "wheat": {245, 222, 179, 255},
	"white": {255, 255, 255, 255}, "whitesmoke": {245, 245, 245, 255}, "yellow": {255, 255, 0, 255},
	"yellowgreen": {154, 205, 50, 255},
}
