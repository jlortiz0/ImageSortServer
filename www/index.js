function FolderMenu(props) {
    const listData = props.folders.map((x, i) => {
        return (
            <li className="w3-hover-dark-gray" onClick={() => props.onClick(i)} key={x}>
                {x} <span onClick={() => props.rmFldr(i)}
                    className="w3-button rmButton">&times;</span>
            </li>
        );
    });
    const len = props.folders.length;
    return (<div>
        {listData}
        <li className="w3-hover-dark-gray" key="Sort" onClick={() => props.onClick(len)}>Sort</li>
        <li className="w3-hover-dark-gray" key="Trash" onClick={() => props.onClick(len + 1)}>Trash
            <span onClick={() => props.rmFldr(len + 1)} className="w3-button rmButton">
                &times;</span></li>
        <li className="w3-hover-dark-gray" key="New" onClick={() => props.onClick(len + 2)}>New...</li>
        <li className="w3-hover-dark-gray" key="Options" onClick={() => props.onClick(len + 3)}>Options</li>
    </div>);
}

function FolderMenuMngr(props) {
    return ReactDOM.createPortal(<FolderMenu folders={props.folders} onClick={props.onClick} rmFldr={props.rmFldr} />,
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
        return ReactDOM.createPortal(<video id="smallImg" src={props.sel} autoPlay controls loop muted playsInline disablePictureInPicture onAnimationEnd={props.animEnd} />,
            document.getElementById("smallImgWrapper"));
    }
    const clss = props.flags & flagsEnum.animUp ? "animUp" : (props.flags & flagsEnum.animDown ? "animDown" : "")
    return ReactDOM.createPortal(<img id="smallImg" className={clss} src={props.sel} onAnimationEnd={props.animEnd} />,
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

// TODO: Consider disabling B on empty folder bar
function ButtonsMngr(props) {
    return ReactDOM.createPortal(<div>
        <button className="w3-button w3-gray w3-right title-bar"
            onClick={props.delAction} disabled={props.curFldr == "Trash"}>D</button>
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
    </div>, document.getElementById("infoModal"));
}

function FolderBar(props) {
    const data = props.folders.map((x, i) => {
        return (
            <button className="w3-btn w3-ripple w3-aqua" onClick={() => props.onClick(i)} key={x}>{x}</button>
        );
    });
    return ReactDOM.createPortal(data, document.getElementById("folderBarMountPoint"));
}

// TODO: Am I going to do anything with the first 3?
const flagsEnum = {
    reserved1: 1,
    reserved: 2,
    reserved2: 4,
    animUp: 8,
    animDown: 16
}

class GodObject extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            folders: [],
            curFldr: "",
            listing: [],
            flags: 0,
            lsind: 0,
            isDiff: false,
            lastMoveLeft: false,
            folderBarVisible: false,
        }
        this.populateFldrList();
    }

    populateFldrList() {
        const loader = new XMLHttpRequest();
        loader.onload = (function () {
            if (loader.status != 200) {
                if (loader.responseText.length != 0) {
                    document.getElementById("errorModalInner").innerHTML = (
                        <p>Error loading folder list:<br />{loader.responseText.toString()}</p>
                    );
                    document.getElementById("errorModal").style.display = "block";
                }
                return
            }
            const ret = JSON.parse(loader.responseText);
            this.setState({
                folders: ret,
            })
        }.bind(this));
        loader.open("GET", "/api/1/list");
        loader.send();
    }

    handleFldrMenuClick(i) {
        document.getElementById('sidebar').style.display = 'none';
        if (i >= this.state.folders.length) {
            switch (i - this.state.folders.length) {
                case 0:
                    this.populateFileList("Sort");
                    document.getElementById('title-text').innerText = "Sort";
                    break;
                case 1:
                    this.populateFileList("Trash");
                    document.getElementById('title-text').innerText = "Trash";
                    break;
                case 2:
                // TODO: New...
                case 3:
                // TODO: Options
            }
            return;
        }
        if (this.state.folders[i] != this.state.curFldr) {
            this.populateFileList(this.state.folders[i]);
            document.getElementById('title-text').innerText = this.state.folders[i];
        }
    }

    populateFileList(fldr) {
        const loader = new XMLHttpRequest();
        loader.onload = (function () {
            if (loader.status != 200) {
                if (loader.responseText.length != 0) {
                    document.getElementById("errorModalInner").innerHTML =
                        "Error loading folder:<br />" + loader.responseText.toString();
                    document.getElementById("errorModal").style.display = true;
                }
                this.setState({
                    flags: 0,
                    curFldr: fldr,
                    listing: [],
                    isDiff: false,
                    folderBarVisible: false,
                })
                return
            }
            const ret = JSON.parse(loader.responseText);
            this.setState({
                listing: ret,
                curFldr: fldr,
                isDiff: false,
                flags: 0,
                lsind: 0,
            }, () => {
                this.addToInd(0);
                this.toggleBar();
            });
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
            if (newInd == -1) {
                return;
            }
        }
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
                if (elem == null) {
                    return;
                }
                this.setState({
                    modalDims: elem.width + "x" + elem.height,
                });
            }.bind(this), 20);
        }.bind(this, i));
        loader.open("GET", "/api/1/info/" + this.state.curFldr + "/" + this.state.listing[newInd]);
        loader.send();
    }

    handleAnimEnd(e) {
        this.setState({
            listing: this.state.newLs,
            newLs: undefined,
            flags: this.state.flags & ~(flagsEnum.animUp | flagsEnum.animDown),
        }, () => this.addToInd(this.state.lastMoveLeft ? -1 : 0));
    }

    delCur() {
        const ind = this.state.lsind;
        const loader = new XMLHttpRequest();
        loader.open("DELETE", "/" + this.state.curFldr + "/" + this.state.listing[ind]);
        loader.send();
        const newLs = this.state.listing.slice();
        newLs.copyWithin(ind, ind + 1);
        newLs.length--;
        this.setState({
            newLs: newLs,
            flags: flagsEnum.animDown,
        });
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
            newLs: newLs,
            flags: flagsEnum.animUp,
        });
    }

    // TODO: diff mode
    diffSwap() {
        if (!this.state.isDiff) {
            return
        }
    }

    toggleBar() {
        if (this.state.curFldr == "Sort" && !this.state.folderBarVisible) {
            this.setState({
                folderBarVisible: true,
            });
            document.getElementById("folderBarMountPoint").style.display = "block";
        } else {
            this.setState({
                folderBarVisible: false,
            });
            document.getElementById("folderBarMountPoint").style.display = "none";
        }
    }

    rmFldr(i) {
        const loader = new XMLHttpRequest();
        loader.onload = function () {
            // if (loader.status != 200) {
            //     return
            // }
            // We should probably reload if there was an error
            // FIXME?: Clicking the X button will cause a folder switch, should probably fix that
            if (i < this.state.folders.length) {
                this.populateFldrList();
            }
        }.bind(this, i);
        if (i == this.state.folders.length + 1) {
            loader.open("DELETE", "/Trash");
        } else if (i >= this.state.folders.length) {
            return;
        } else {
            loader.open("DELETE", "/" + this.state.folders[i]);
        }
        loader.send();
    }

    // TODO: Keyboard controls
    render() {
        const sel = this.state.listing.length ? "/" + this.state.curFldr + "/" + this.state.listing[this.state.lsind] : "empty.svg";
        return (<div>
            <FolderBar folders={this.state.folders} onClick={(i) => this.moveCur(this.state.folders[i])} />
            <LargeImageMngr sel={sel} isVideo={isVideo(sel)} />
            <SmallImageMngr sel={sel} isVideo={isVideo(sel)} animEnd={(e) => this.handleAnimEnd(e)} flags={this.state.flags} />
            <ButtonsMngr curFldr={this.state.curFldr} isDiff={this.state.isDiff} sel={this.state.lsind}
                max={this.state.listing.length} laction={() => this.addToInd(-1)} raction={() => this.addToInd(1)}
                sortAction={this.state.curFldr == "Sort" ? () => this.toggleBar() : () => this.moveCur("Sort")}
                delAction={() => this.delCur()} switchAction={() => this.diffSwap()} dims={this.state.modalDims} />
            <FolderMenuMngr folders={this.state.folders} onClick={(i) => this.handleFldrMenuClick(i)} rmFldr={(i) => this.rmFldr(i)} flags={this.state.flags} />
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
