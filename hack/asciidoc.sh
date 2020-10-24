#!/usr/bin/env bash
​
MYBASEDIR="$(pwd)"
​
# Get upstream book version
git clone https://github.com/fmuyassarov/metal3-docs
pushd metal3-docs
​
# Run the upstream book grabber for all the modules
sh metal3-docs/build.sh --justgrab

​popd

# Replace the upstream book folder with our own PR content, available at ./
cd ${MYBASEDIR}
​
#FIXME replace with command to get this current repo name
MYEPONAME=$()
​
rsync -avr --progress --delete  --exclude 'fullmetaljacket' ./ fullmetaljacket/${MYREPONAME}
​
# Now, the PR has the upstream repos for other things, and the content that has changed in this PR overwriting what was in 'master'
​
​
#FIXME (do whatever is needed to build the full book)
sh metal3-docs/build.sh --justbuild