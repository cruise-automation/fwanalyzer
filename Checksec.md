# checksec Integration

[checksec](https://github.com/slimm609/checksec.sh) is a bash script for checking security properties of executables (like PIE, RELRO, Canaries, ...).

Checksec is an incredible helpful tool therefore we developed a wrapper script for FwAnalyzer to ease the usage of checksec. Below
we go through the steps required to use checksec with FwAnalyzer.

## Installation

The installation is rather simple. Clone the checksec repository and copy the `checksec` script to a directory in your PATH
or add the directory containing `checksec` to your PATH.

## Configuration

Configuration is done in two steps. First step is adding a `FileContent` check that uses the `Script` option.
The second step is creating the checksec wrapper configuration. The configuration allows you to selectively skip files
(e.g. vendor binaries) and fine tune the security features that you want to enforce.

### checksec wrapper configuration

The checksec wrapper has two options, and uses JSON:

- cfg : checksec config, where you can select acceptable values for each field in the checksec output. The key is the name of the checksec field and the value is an array where each item is an acceptable value (e.g. allow `full` and `partial` RELRO). Omitted fields are not checked.
- skip : array of fully qualified filenames that should be not checked

example config:
```json
{
  "cfg":
  {
    "pie": ["yes"],
    "nx": ["yes"],
    "relo": ["full", "partial"]
  },
  "skip": ["/usr/bin/bla","/bin/blabla"]
}
```

### FwAnalyzer configuration

The FwAnalyzer configuration uses the checksec wrapper config and looks like in the example below.
We define a `FileContent` check and select `/usr/bin` as the target directory.
The name of the wrapper script is `check_sec.sh`.
We pass two options to the script. First argument `*` selects all files in `/usr/bin` and
the second argument is the checksec wrapper config we created above.

example config:
```ini
[FileContent."checksec_usr_bin"]
File = "/usr/bin"
Script = "check_sec.sh"
ScriptOptions = ["*",
"""
{
"cfg":{
  "pie": ["yes"],
  "nx": ["yes"],
  "relo": ["full", "partial"]
 },
 "skip": ["/usr/bin/bla","/bin/blabla"]
}
"""]
```


### Example Output

```json
"offenders": {
  "/usr/bin/example": [
  {
    "canary": "no",
    "fortified": "0",
    "fortify-able": "24",
    "fortify_source": "no",
    "nx": "yes",
    "pie": "no",
    "relro": "partial",
    "rpath": "no",
    "runpath": "no",
    "symbols": "no"
  }
  ]
}
```
