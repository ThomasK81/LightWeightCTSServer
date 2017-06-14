# LightWeightCTSServer
A light weight Server for Perseus CTS XML files written in Go. This is a potentially buggy alpha-version. 

## Get Started
1. Download the zipped Repo
2. Unpack it (You can rename it afterwards if you like)
3. Open a Terminal/Commandline and cd into the unpacked (and optionally renamed) folder

4. On Mac/Linux: start it up with `./UnixCTS_V002` / on Windows `WindowsCTS_V002.exe` 

## Trouble-shooting

On OSX/Linux you might have to tell your Operating system that `./UnixCTS_V002` is an executable with `chmod +x UnixCTS_V002`

## Try it with your favourite browser

1. Whole work: http://localhost:8000/cts/full/tlg0003.tlg001.perseus-grc2/
2. Part of it: http://localhost:8000/cts/chunk/phi0448.phi001.perseus-lat2:1.1.2
3. Range: http://localhost:8000/cts/range/tlg0085.tlg005.opp-grc3:1-3
4. The Range command supports substrings in unicode: http://localhost:8000/cts/range/tlg0085.tlg005.opp-grc3:1@τῶνδ᾽-12@ἂν 

**!! Please take not of the `/full/` `/chunk/` `/range/` part of the http link !!**

## Alpheios Dictionaries

Alpheios' dictionaries are now implemented. **Tested in Chrome/Firefox/Safari. Works in Chrome and Firefox. Does not work in Safari.**
![Alpheios Dictionary](https://raw.githubusercontent.com/ThomasK81/LightWeightCTSServer/master/Alpheios.png)

## Adding your own CTS XML

1. Download your favourite Leipzig/Perseus CTS XML file (for a selection have a look here: http://opengreekandlatin.github.io/First1KGreek/)

2. Copy it into the unzipped folder into the subfolder `static/OPP` (you can use Finder/Eplorer for this and drag&drop)
3. Enjoy your text under http://localhost:8080/cts/full/ + [The FILENAME without.XML] and query the text as outlined above

Here is video that demonstrates how easy it is to integrate new XML file: https://drive.google.com/file/d/0BzNW0LZy0RUOLU5jc2dEOE8ybHc/view?usp=sharing

## Modify Port and Source for XML and other stuff (for non-technical people point 3 is of interest).

1. config.json is pretty much self-explicable. You can point the app to a web resource where CTS XML text are stored if you like though.
2. You can change the look and feel of your website using HTML and CSS 
3. The webpage connects to hypothes.is you can annotate any text and share notes publically if you like
