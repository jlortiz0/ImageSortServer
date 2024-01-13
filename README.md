# ImageSortServer - ImageSort in React

This program was written in a rather short amount of time so that I could learn React in order to complete an [college project](https://github.com/tristan8chu/Just-Move-115A). Turns out I was using a very outdated tutorial that taught class based components. Also, I directly ported the paradigms from [ImageSort](https://github.com/jlortiz0/ImageSort) instead of reconsidering things in a more structured way, resulting in a god object and use of portals.

And of course, I wrote a custom Go server that reinvents the FileServer Handler instead of using Node. In my defense, I had previously had negative experiences with installing and using Node. Thankfully work on the project convinced me of its usefulness.

I never intended to maintain or even use this program, and as a result it has various bugs (like the image dimensions never being correct) and incomplete functions (such as keyboard controls).

My initial design for the API can be found in ImgSrtSrvSpec.txt