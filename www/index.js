function FolderMenu(props) {
    const listData = props.folders.map((x, i) => {
        return (
            <li className="w3-hover-dark-gray" onClick={() => props.onClick(i)} key={x}>
                {x} <span onClick={(e) => { e.stopPropagation(); props.rmFldr(i); }}
                    className="w3-button rmButton">&times;</span>
            </li>
        );
    });
    const len = props.folders.length;
    const trashClss = props.trashGreen ? "w3-hover-green" : "w3-hover-dark-gray";
    return ReactDOM.createPortal(<div>
        {listData}
        <li className="w3-hover-dark-gray" key="Sort" onClick={() => props.onClick(len)}>Sort</li>
        <li className={trashClss} key="Trash" onClick={() => props.onClick(len + 1)}>Trash
            <span onClick={(e) => { e.stopPropagation(); props.rmFldr(len + 1); }} className="w3-button rmButton">
                &times;</span></li>
        <li className="w3-hover-dark-gray" key="New" onClick={() => props.onClick(len + 2)}>New...</li>
        <li className="w3-hover-dark-gray" key="Options" onClick={() => props.onClick(len + 3)}>Options</li>
    </div>, document.getElementById("folderMenuMountPoint"));
}

function LargeImageMngr(props) {
    if (props.isVideo) {
        return;
    }
    return ReactDOM.createPortal(<img id="bigImg" src={props.sel} />,
        document.getElementById("imageModal"));
}

function SmallImageMngr(props) {
    const clss = props.flags & flagsEnum.animUp ? "animUp" : (props.flags & flagsEnum.animDown ? "animDown" : "")
    if (props.isVideo) {
        return ReactDOM.createPortal(<video id="smallImg" className={clss} src={props.sel} autoPlay controls loop muted playsInline disablePictureInPicture onAnimationEnd={props.animEnd} />,
            document.getElementById("smallImgWrapper"));
    }
    return ReactDOM.createPortal(<img id="smallImg" className={clss} src={props.sel} onAnimationEnd={props.animEnd} />,
        document.getElementById("smallImgWrapper"));
}

function LRButtonsMngr(props) {
    const ds = props.isDiff ? (props.diffWhich ? "F2 " : "F1 ") : "";
    return ReactDOM.createPortal(<div>
        <button className="w3-btn w3-display-left w3-white ui-layer2" onClick={props.laction} disabled={!props.sel}>&lt;</button>
        <button className="w3-btn w3-display-right w3-white ui-layer2" onClick={props.raction} disabled={props.sel + 1 >= props.max}>&gt;</button>
        <span className="w3-text-black w3-white ui-layer2" id="dimLabel">{props.dims}</span>
        <span className="w3-text-black w3-white ui-layer2" id="indLabel">
            {ds}<input id="gotoInput" onBlur={props.gaction} defaultValue={props.sel + 1} type="number" min="1" max={props.max} onKeyDown={function (e) {
                const elem = document.getElementById("gotoInput");
                if (e.key == "Enter") {
                    elem.blur();
                } else if (e.key == "Escape") {
                    elem.value = "";
                    elem.blur();
                }
            }} />
            /{props.max}</span>
    </div>, document.getElementById("lrButtonsMountPoint"));
}

function ButtonsMngr(props) {
    return ReactDOM.createPortal(<div>
        <button className="w3-button w3-gray w3-right title-bar ui-layer2"
            onClick={props.delAction} disabled={props.curFldr == "Trash"}>D</button>
        <button className="w3-button w3-gray w3-right title-bar ui-layer2"
            onClick={props.sortAction}>{props.curFldr == "Sort" ? "B" : "S"}</button>
        <button className="w3-button w3-gray w3-right title-bar ui-layer2"
            onClick={() => document.getElementById('infoModal').style.display = 'block'}>I</button>
        <button className="w3-button w3-gray w3-right title-bar ui-layer2"
            onClick={props.switchAction}>O</button>
        <LRButtonsMngr sel={props.sel} laction={props.laction} raction={props.raction} max={props.max} dims={props.dims}
            isDiff={props.isDiff} diffWhich={props.diffWhich} gaction={props.gaction} />
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

function NewFldrModal(props) {
    return ReactDOM.createPortal(<button className="w3-btn w3-blue" onClick={props.onClick}>Create</button>, document.getElementById("fldrModalMountPoint"));
}

function SettingsModal(props) {
    return ReactDOM.createPortal(<button className="w3-btn w3-blue" onClick={props.onClick}>Save</button>, document.getElementById("settingsModalMountPoint"));
}

const flagsEnum = {
    animUp: 1,
    animDown: 2,
    trashGreen: 4
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
            diffWhich: false,
        }
        window.addEventListener('keydown', (e) => this.handleKey(e), true);
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
                    document.getElementById("newFldrModal").style.display = "block";
                    document.getElementById("newFldrInput").focus();
                    break;
                case 3:
                    const loader = new XMLHttpRequest();
                    loader.onload = function () {
                        const ret = JSON.parse(loader.responseText);
                        document.getElementById("settingHashSize").value = ret.HashSize.toString();
                        document.getElementById("settingHashDiff").value = ret.HashDiff.toString();
                        document.getElementById("settingsModal").style.display = "block";
                    }
                    loader.open("GET", "/api/1/settings");
                    loader.send();
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
        // Seems it isn't possible to get a list of supported mime types...
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
                diffWhich: false,
            });
            setTimeout(function () {
                document.getElementById("indLabel").children[0].value = this.state.lsind + 1;
                const elem = document.getElementById("bigImg");
                if (elem == null) {
                    return;
                }
                elem.scrollIntoView({
                    behavior: "auto",
                    block: "center",
                    inline: "center",
                });
                this.setState({
                    modalDims: elem.width + "x" + elem.height,
                });
            }.bind(this), 20);
        }.bind(this, i));
        loader.open("GET", "/api/1/info/" + this.state.curFldr + "/" + (this.state.isDiff ? this.state.listing[newInd][0] : this.state.listing[newInd]));
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
        loader.open("DELETE", "/" + this.state.curFldr + "/" + (this.state.isDiff ? this.state.listing[ind][0] : this.state.listing[ind]));
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
        loader.open("POST", "/" + this.state.curFldr + "/" + (this.state.isDiff ? this.state.listing[ind][0] : this.state.listing[ind]));
        loader.send(loc);
        const newLs = this.state.listing.slice();
        newLs.copyWithin(ind, ind + 1);
        newLs.length--;
        this.setState({
            newLs: newLs,
            flags: flagsEnum.animUp,
        });
    }

    beginDiffMode() {
        const loader = new XMLHttpRequest();
        const doit = function (_, loader) {
            if (loader.status == 202) {
                const loader2 = new XMLHttpRequest();
                loader2.onload = () => doit(loader2);
                loader2.open("GET", "/api/1/dedup?token=" + loader.responseText);
                setTimeout(() => loader2.send(), 2500);
                return
            } else if (loader.status != 200) {
                return;
            }
            const ls = JSON.parse(loader.responseText);
            this.setState({
                isDiff: true,
                diffWhich: false,
                listing: ls,
                curFldr: ".",
                lsind: 0,
            }, () => {
                document.getElementById("title-text").innerText = "DeDuplicator";
                this.addToInd(0);
            });
        }.bind(this, doit);
        loader.onload = (() => doit(loader)).bind(loader);
        loader.open("GET", "/api/1/dedup");
        loader.send();
    }

    diffSwap() {
        if (!this.state.isDiff) {
            this.beginDiffMode()
            return
        }
        this.setState({
            diffWhich: !this.state.diffWhich,
        });
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
            // We should probably reload if there was an error
            if (i < this.state.folders.length) {
                this.populateFldrList();
            } else {
                this.setState({
                    flags: flagsEnum.trashGreen,
                });
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

    handleNewFldr() {
        const v = document.getElementById("newFldrInput").value;
        document.getElementById("newFldrInput").value = "";
        if (v.trim() == "") {
            return;
        }
        document.getElementById("newFldrModal").style.display = "none";
        const loader = new XMLHttpRequest();
        loader.open("CREATE", "/" + v);
        loader.onload = () => this.populateFldrList();
        loader.send();
    }

    handleSettingsSave() {
        const loader = new XMLHttpRequest();
        const body = {
            HashSize: parseInt(document.getElementById("settingHashSize").value),
            HashDiff: parseInt(document.getElementById("settingHashDiff").value),
        };
        loader.onload = () => document.getElementById("settingsModal").style.display = "block";
        loader.open("PUT", "/api/1/settings");
        loader.send(JSON.stringify(body));
    }

    handleGoto() {
        const v = document.getElementById("gotoInput");
        const i = parseInt(v.value);
        if (i != NaN && v.value != "") {
            this.addToInd(i - this.state.lsind - 1);
        } else {
            v.value = this.state.lsind + 1;
        }
        window.focus();
    }

    handleKey(e) {
        if (e.defaultPrevented) {
            return;
        }
        switch (e.key) {
            case "Left":
            case "ArrowLeft":
                if (this.state.lsind) {
                    this.addToInd(-1);
                }
                break;
            case "Right":
            case "ArrowRight":
                if (this.state.lsind < this.state.listing.length) {
                    this.addToInd(1);
                }
                break;
            case "Home":
                this.setState({ lsind: 0 }, () => this.addToInd(0));
                break;
            case "End":
                this.setState({ lsind: this.state.listing.length - 1 }, () => this.addToInd(0));
                break;
            case "c":
                this.delCur();
                break;
            case "x":
                if (this.state.curFldr != "Sort") {
                    this.moveCur("Sort");
                }
                break;
            case "z":
                const elem = document.getElementById('infoModal');
                if (elem.style.display == "none") {
                    elem.style.display = "flex";
                } else {
                    elem.style.display = "none";
                }
                break;
            case "g":
                document.getElementById("gotoInput").focus();
                break;
            case "q":
                if (this.state.isDiff) {
                    this.diffSwap();
                }
                break;
            case "Up":
            case "ArrowUp":
                document.getElementById("bigImg").style.display = "block";
                break;
            case "Down":
            case "ArrowDown":
                document.getElementById("bigImg").style.display = "none";
                break;
            case "w":
                document.getElementById("imageModal").scrollBy(0, -10);
                break;
            case "s":
                document.getElementById("imageModal").scrollBy(0, 10);
                break;
            case "a":
                document.getElementById("imageModal").scrollBy(-10, 0);
                break;
            case "d":
                document.getElementById("imageModal").scrollBy(10, 0);
                break;
            // TODO: folder bar controls, maybe switch up/down to v
            default:
                return;
        }
        e.preventDefault();
    }

    render() {
        const sel2 = this.state.listing.length ? (this.state.isDiff ? this.state.listing[this.state.lsind][this.state.diffWhich ? 1 : 0] : this.state.listing[this.state.lsind]) : undefined;
        const sel = this.state.listing.length ? "/" + this.state.curFldr + "/" + sel2 : "empty.svg";
        return (<div>
            <FolderBar folders={this.state.folders} onClick={(i) => this.moveCur(this.state.folders[i])} />
            <LargeImageMngr sel={sel} isVideo={isVideo(sel)} />
            <SmallImageMngr sel={sel} isVideo={isVideo(sel)} animEnd={(e) => this.handleAnimEnd(e)} flags={this.state.flags} />
            <ButtonsMngr curFldr={this.state.curFldr} isDiff={this.state.isDiff} sel={this.state.lsind} diffWhich={this.state.diffWhich}
                max={this.state.listing.length} laction={() => this.addToInd(-1)} raction={() => this.addToInd(1)}
                sortAction={this.state.curFldr == "Sort" ? () => this.toggleBar() : () => this.moveCur("Sort")}
                delAction={() => this.delCur()} switchAction={() => this.diffSwap()} dims={this.state.modalDims} gaction={() => this.handleGoto()} />
            <FolderMenu folders={this.state.folders} onClick={(i) => this.handleFldrMenuClick(i)} rmFldr={(i) => this.rmFldr(i)} flags={this.state.flags} trashGreen={this.state.flags & flagsEnum.trashGreen} />
            <InfoModal size={this.state.modalSize} fName={this.state.listing.length ? sel2 : "empty.svg"} />
            <NewFldrModal onClick={() => this.handleNewFldr()} />
            <SettingsModal onClick={() => this.handleSettingsSave()} />
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
