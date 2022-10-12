function FolderMenu(props) {
    return props.folders.map((x, i) => {
        return (
            <li className="w3-hover-dark-gray" onClick={() => props.onClick(i)} key={x}>{x}</li>
        );
    })
}

function FolderMenuMngr(props) {
    return ReactDOM.createPortal(<FolderMenu folders={props.folders} onClick={props.onClick} />,
        document.getElementById("folderMenuMountPoint"));
}

function LargeImageMngr(props) {
    if (props.isVideo) {
        return;
    }
    return ReactDOM.createPortal(<img id="bigImg" src={props.sel} />,
        document.getElementById("imageModal"));
}

function SmallImageMngr(props) {
    if (props.isVideo) {
        return ReactDOM.createPortal(<video id="smallImg" src={props.sel} autoPlay controls loop muted playsInline disablePictureInPicture />,
            document.getElementById("smallImgWrapper"));
    }
    return ReactDOM.createPortal(<img id="smallImg" src={props.sel} />,
        document.getElementById("smallImgWrapper"));
}

function LRButtonsMngr(props) {
    return ReactDOM.createPortal(<div>
        <button className="w3-btn w3-display-left w3-white ui-layer2" onClick={props.laction} disabled={!props.sel}>&lt;</button>
        <button className="w3-btn w3-display-right w3-white ui-layer2" onClick={props.raction} disabled={props.sel + 1 >= props.max}>&gt;</button>
        <span className="w3-text-black w3-white ui-layer2" id="dimLabel">{props.dims}</span>
        <span className="w3-text-black w3-white ui-layer2" id="indLabel">{props.sel + 1}/{props.max}</span>
    </div>, document.getElementById("lrButtonsMountPoint"));
}

function ButtonsMngr(props) {
    return ReactDOM.createPortal(<div><button className="w3-button w3-gray w3-right title-bar"
        onClick={props.delAction}>D</button>
        <button className="w3-button w3-gray w3-right title-bar"
            onClick={props.sortAction}>{props.curFldr == "Sort" ? "B" : "S"}</button>
        <button className="w3-button w3-gray w3-right title-bar"
            onClick={() => document.getElementById('infoModal').style.display = 'block'}>I</button>
        <button className="w3-button w3-gray w3-right title-bar" disabled={!props.isDiff}
            onClick={props.switchAction}>O</button>
        <LRButtonsMngr sel={props.sel} laction={props.laction} raction={props.raction} max={props.max} dims={props.dims} />
    </div>, document.getElementById("buttonsMountPoint"));
}

function InfoModal(props) {
    var sizeDisplay;
    if (props.size > 1024 * 1024) {
        sizeDisplay = (props.size / (1024 * 1024)).toFixed(1) + " MiB";
    } else if (props.size > 1024) {
        sizeDisplay = (props.size / 1024).toFixed(1) + " KiB";
    } else {
        sizeDisplay = props.size + " B";
    }
    return ReactDOM.createPortal(<div className="w3-modal-content w3-white w3-display-middle">
        <p>Image: {props.fName}</p>
        <p>Storage: {sizeDisplay}</p>
        {/* <p>Dimensions: {props.dims}</p> */}
    </div>, document.getElementById("infoModal"));
}

const flagsEnum = {
    loadingFldrList: 1,
    loadingImg: 2,
    loadingFldr: 4,
}

class GodObject extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            folders: [],
            curFldr: "",
            listing: [],
            flags: flagsEnum.loadingFldrList,
            lsind: 0,
            isDiff: false,
            lastMoveLeft: false,
        }
        this.populateFldrList();
    }

    populateFldrList() {
        const loader = new XMLHttpRequest();
        loader.onload = (function () {
            if (loader.status != 200) {
                if (loader.responseText.length != 0) {
                    document.getElementById("errorModalInner").innerHTML = (
                        <p>Error loading folder list:<br />{loader.responseText}</p>
                    );
                }
                this.setState({
                    flags: 0,
                })
                return
            }
            const ret = JSON.parse(loader.responseText);
            this.setState({
                folders: ret,
                flags: 0,
            })
        }.bind(this));
        loader.open("GET", "/api/1/list");
        loader.send();
    }

    handleFldrMenuClick(i) {
        if (this.state.folders[i] != this.state.curFldr) {
            this.setState({
                flags: flagsEnum.loadingFldr,
            });
            this.populateFileList(this.state.folders[i]);
            document.getElementById('title-text').innerText = this.state.folders[i];
        }
        document.getElementById('sidebar').style.display = 'none';
    }

    populateFileList(fldr) {
        const loader = new XMLHttpRequest();
        loader.onload = (function () {
            if (loader.status != 200) {
                if (loader.responseText.length != 0) {
                    document.getElementById("errorModalInner").innerHTML =
                        "Error loading folder:<br />" + loader.responseText;
                    document.getElementById("errorModal").style.display = true;
                }
                this.setState({
                    flags: 0,
                    curFldr: fldr,
                    listing: [],
                    isDiff: false,
                })
                return
            }
            const ret = JSON.parse(loader.responseText);
            this.setState({
                listing: ret,
                curFldr: fldr,
                isDiff: false,
                lsind: 0,
            });
            // I hate timings
            setTimeout(() => this.addToInd(0), 20);
        }.bind(this, fldr));
        loader.open("GET", "/api/1/list/" + fldr);
        // TODO: Smarter way of finding supported images
        loader.setRequestHeader("Accept", "image/jpeg,image/png,image/bmp,image/gif,image/webp,video/mp4,video/ogg,video/mpeg,video/webm");
        loader.send();
    }

    addToInd(i) {
        const lastMoveLeft = i < 0;
        var newInd = this.state.lsind + i;
        if (newInd < 0) {
            newInd = 0;
        } else if (newInd >= this.state.listing.length) {
            newInd = this.state.listing.length - 1;
        }
        this.setState({
            flags: flagsEnum.loadingImg,
        });
        const loader = new XMLHttpRequest();
        loader.onload = (function () {
            if (loader.status != 200) {
                const newLs = this.state.listing.slice();
                newLs.copyWithin(newInd, newInd + 1);
                newLs.length--;
                this.setState({
                    listing: newLs,
                    lastMoveLeft: lastMoveLeft,
                });
                this.addToInd(i + (this.state.lastMoveLeft ? -1 : 0));
                return;
            }
            this.setState({
                flags: 0,
                lsind: newInd,
                modalSize: parseInt(loader.responseText, 10),
                lastMoveLeft: lastMoveLeft,
            });
            setTimeout(function () {
                const elem = document.getElementById("bigImg");
                this.setState({
                    modalDims: elem.width + "x" + elem.height,
                });
            }.bind(this), 20);
        }.bind(this, i));
        loader.open("GET", "/api/1/info/" + this.state.curFldr + "/" + this.state.listing[newInd]);
        loader.send();
    }

    // TODO: These two functions bounce the slice but not the request. Figure out why.
    delCur() {
        const ind = this.state.lsind;
        const loader = new XMLHttpRequest();
        loader.open("DELETE", "/" + this.state.curFldr + "/" + this.state.listing[ind]);
        loader.send();
        const newLs = this.state.listing.slice();
        newLs.copyWithin(ind, ind + 1);
        newLs.length--;
        this.setState({
            listing: newLs,
        });
        setTimeout(() => this.addToInd(this.state.lastMoveLeft ? -1 : 0), 20);
    }

    moveCur(loc) {
        const ind = this.state.lsind;
        const loader = new XMLHttpRequest();
        loader.open("POST", "/" + this.state.curFldr + "/" + this.state.listing[ind]);
        loader.send(loc);
        const newLs = this.state.listing.slice();
        newLs.copyWithin(ind, ind + 1);
        newLs.length--;
        this.setState({
            listing: newLs,
        });
        setTimeout(() => this.addToInd(this.state.lastMoveLeft ? -1 : 0), 20);
    }

    // TODO: diff mode
    diffSwap() {
        if (!this.state.isDiff) {
            return
        }
    }

    // TODO: folder bar
    toggleBar() {

    }

    // TODO: Keyboard controls
    // TODO: folder delete, trash clear
    render() {
        const sel = this.state.listing.length ? "/" + this.state.curFldr + "/" + this.state.listing[this.state.lsind] : "empty.svg";
        return (<div>
            <LargeImageMngr sel={sel} isVideo={isVideo(sel)} />
            <SmallImageMngr sel={sel} isVideo={isVideo(sel)} />
            <ButtonsMngr curFldr={this.state.curFldr} isDiff={this.state.isDiff} sel={this.state.lsind}
                max={this.state.listing.length} laction={() => this.addToInd(-1)} raction={() => this.addToInd(1)}
                sortAction={this.state.curFldr == "Sort" ? () => this.toggleBar() : () => this.moveCur("Sort")}
                delAction={() => this.delCur()} switchAction={() => this.diffSwap()} dims={this.state.modalDims} />
            <FolderMenuMngr folders={this.state.folders} onClick={(i) => this.handleFldrMenuClick(i)} />
            <InfoModal size={this.state.modalSize} fName={this.state.listing.length ? this.state.listing[this.state.lsind] : "empty.svg"} />
        </div>
        );
    }
}

const root = ReactDOM.createRoot(document.getElementById("reactRoot"));
root.render(<GodObject />);

function isVideo(path) {
    const ind = path.indexOf(".");
    if (ind == -1) {
        return false;
    }
    const ext = path.slice(ind + 1);
    switch (ext) {
        case "mp4":
            return true;
        case "webm":
            return true;
        case "ogv":
            return true;
        case "mpeg":
            return true;
    }
    return false;
}
