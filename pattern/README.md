# go-lovense/pattern

## Notes

File format:

```
V:1;T:Edge;F:v1,v2;S:100;M:...;#
0,0;0,0;0,0;0,0;0,0;0,0;0,0;0,0;0,0;0,0;0,0;0,0;0,0;0,0;0,0;...
```

- `V` is presumably the version. There doesn't seem to be code that checks for
  it, but presumably, it's just assumed to be `1`.
- `F` seems to be the motor:

	```js
    ("p" === k
      ? w.push("Air:Level")
      : "r" === k
      ? w.push("Rotate")
      : "v" === k
      ? w.push("Vibrate")
      : "v1" === k
      ? w.push("Vibrate1")
      : "v2" === k && w.push("Vibrate2"));
	```

  - The Edge 2 has `v1,v2`, probably indicating 2 of its motors.

- The source code seems to have `toyOnlineImg`, which can be useful for getting
  the image URL.
- `M` is the MD5 hex digest of something. The code involves `Math.random`, so
  it's quite unclear. The routine `generatePattern` has this.
- `S` is hard-coded to 100 in the `generatePattern` routine.
- The final `#` seems to denote the separator for the metadata and the vibration
  data. The new lines are replaced out, and data are separated by `;`.
- The routine `e.resetPatternTime` is used for converting seconds to `MM:SS`.
  Tracing where it's called might be a good idea.
  - The code splits `0,0` in the data line into `data0` and `data1`. The
    duration is calculated by doing

	  resetPatternTime(100 * data0.length / 1000)
	= resetPatternTime(data0.length / 10)

  - The code also hard-codes a variable `orderSpeed = 100`. This seems to be
    used as the argument for `setTimeout()`. Presumably, this implies that each
	point in the file is executed within 100ms or 10Hz.
  - The code sets `orderSpeed` to `patternData.data.S` later on, however,
  presumably meaning that the `S` variable overrides the duration.
