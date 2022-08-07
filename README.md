## This is attempt to create video player using Golang
Plays .mp4, .mkv files

### Based on:
*  https://github.com/zergon321/reisen — for decoding media files
*  OpenGL — for rendering video

### Usage
1. Start executable from directory you put it in
2. To get usage info
```
./videoplayer --help
```
3. To play video
```
./videoplayer --file ./name_of_the_file_with_extension
```
4. Interaction:
   - pause — red button
   - play — green button
   - stop — blue button
   - sound — at the right bottom corner of the video player window

### Known issues
1. Can't decode file if it contains few audio/video streams, subtitles
(because inner library doesn't support this https://github.com/zergon321/reisen)
2. Can't play all files properly. On some files:
   - video/audio is twitching
   - rewinding makes file plays from start

### Todo:
1. Create button icons
2. Make handlers bar dissapears after some time idle (without mouse moving)
3. Add timers (remaining time, elapsed time)
4. Add speaker icon
5. Add fullscreen mode
6. Refactor code structure to be more flexible
7. Resolve issues with not/wrong playing different files
   (change underlying library, fix code bugs)
