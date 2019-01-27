package code

// Built-in operators. In some cases the stack shapes provided are false
// approximations, since some of the built-in operators consume a dynamic
// amount of stack (e.g., "copy", "clear").
//
// This is not a complete list of operators.
var (
	// Stack manipulation
	Clear = Op("clear", 0, 0)
	Count = Op("count", 0, 1)
	Dup   = Op("dup", 1, 2)
	Exch  = Op("exch", 2, 2)
	Index = Op("index", 1, 1)
	Mark  = Op("mark", 0, 1)
	Pop   = Op("pop", 1, 0)
	Roll  = Op("roll", 2, 0)

	// Control
	Exit = Op("exit", 0, 0)

	// Arithmetic
	Abs      = Op("abs", 1, 1)
	Add      = Op("add", 2, 1)
	Atan     = Op("atan", 2, 1)
	Ceiling  = Op("ceiling", 1, 1)
	Div      = Op("div", 2, 1)
	Floor    = Op("floor", 1, 1)
	IDiv     = Op("idiv", 2, 1)
	Mod      = Op("mod", 2, 1)
	Mul      = Op("mul", 2, 1)
	Neg      = Op("neg", 1, 1)
	Round    = Op("round", 1, 1)
	Sqrt     = Op("sqrt", 1, 1)
	Sub      = Op("sub", 2, 1)
	Truncate = Op("truncate", 1, 1)
	Cos      = Op("cos", 1, 1)
	Sin      = Op("sin", 1, 1)
	Exp      = Op("exp", 2, 1)
	Log      = Op("ln", 1, 1)
	Log10    = Op("log", 1, 1)

	// Indexing
	Length = Op("length", 1, 1)
	Get    = Op("get", 2, 1)
	Put    = Op("put", 3, 0)

	// Data structures
	NDict  = Op("dict", 1, 1)
	Begin  = Op("begin", 1, 0)
	End    = Op("end", 0, 0)
	NArray = Op("array", 1, 1)

	// Relations
	And   = Op("and", 2, 1)
	Eq    = Op("eq", 2, 1)
	False = Op("false", 0, 1)
	Ge    = Op("ge", 2, 1)
	Gt    = Op("gt", 2, 1)
	Le    = Op("le", 2, 1)
	Lt    = Op("lt", 2, 1)
	Ne    = Op("ne", 2, 1)
	Not   = Op("not", 1, 1)
	Or    = Op("or", 2, 1)
	Shift = Op("shift", 2, 1)
	True  = Op("true", 0, 1)
	Xor   = Op("xor", 2, 1)

	// Conversions
	ToInt  = Op("cvi", 1, 1)
	ToName = Op("cvn", 1, 1)
	ToReal = Op("cvr", 1, 1)

	// Path construction
	NewPath      = Op("newpath", 0, 0)
	CurrentPoint = Op("currentpoint", 0, 2)
	MoveTo       = Op("moveto", 2, 0)
	RMoveTo      = Op("rmoveto", 2, 0)
	LineTo       = Op("lineto", 2, 0)
	RLineTo      = Op("rlineto", 2, 0)
	Arc          = Op("arc", 5, 0)
	ArcN         = Op("arcn", 5, 0)
	ArcT         = Op("arct", 5, 0)
	ArcTo        = Op("arcto", 5, 4)
	CurveTo      = Op("curveto", 6, 0)
	RCurveTo     = Op("rcurveto", 6, 0)
	ClosePath    = Op("closepath", 0, 0)
	FlattenPath  = Op("flattenpath", 0, 0)
	ReversePath  = Op("reversepath", 0, 0)
	StrokePath   = Op("strokepath", 0, 0)
	ClipPath     = Op("clippath", 0, 0)
	SetBBox      = Op("setbbox", 4, 0)
	PathBBox     = Op("pathbbox", 0, 4)
	InitClip     = Op("initclip", 0, 0)
	Clip         = Op("clip", 0, 0)
	EOClip       = Op("eoclip", 0, 0)
	RectClip     = Op("rectclip", 4, 0)

	// Painting
	ErasePage = Op("erasepage", 0, 0)
	Stroke    = Op("stroke", 0, 0)
	Fill      = Op("fill", 0, 0)
	EOFill    = Op("eofill", 0, 0)

	// Output
	ShowPage = Op("showpage", 0, 0)

	// Fonts and glyphs
	FindFont    = Op("findfont", 1, 1)
	ScaleFont   = Op("scalefont", 2, 1)
	SetFont     = Op("setfont", 1, 0)
	Show        = Op("show", 1, 0)
	StringWidth = Op("stringwidth", 1, 1)

	// Coordinates
	Translate = Op("translate", 2, 0)
	Scale     = Op("scale", 2, 0)
	Rotate    = Op("rotate", 1, 0)

	// Graphics state
	GSave        = Op("gsave", 0, 0)
	GRestore     = Op("grestore", 0, 0)
	SetGray      = Op("setgray", 1, 0)
	SetRGBColor  = Op("setrgbcolor", 3, 0)
	SetLineCap   = Op("setlinecap", 1, 0)
	SetLineJoin  = Op("setlinejoin", 1, 0)
	SetLineWidth = Op("setlinewidth", 1, 0)
)
