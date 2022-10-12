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
    return ReactDOM.createPortal(<img id="bigImg" src={props.sel} />,
        document.getElementById("imageModal"));
}

function SmallImageMngr(props) {
    return ReactDOM.createPortal(<img id="smallImg" src={props.sel} />,
        document.getElementById("smallImgWrapper"));
}

// TODO: Flag consts and proper empty handling, get folder list
const flagsEnum = {
    loadingFldrList: 1,
    loadingImg: 2,
    loadingFldr: 4,
}

class GodObject extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            folders: ["1", "2", "3"],
            curFldr: "",
            listing: [],
            flags: flagsEnum.loadingFldr,
            lsind: 0,
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
                curFldr: this.state.folders[i],
                flags: flagsEnum.loadingFldr,
            });
            this.populateFileList(this.state.folders[i]);
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
                }
                this.setState({
                    flags: 0,
                })
                return
            }
            const ret = JSON.parse(loader.responseText);
            this.setState({
                listing: ret,
                flags: 0,
            })
        }.bind(this));
        loader.open("GET", "/api/1/list/" + fldr);
        // TODO: Smarter way of finding supported images
        loader.setRequestHeader("Accept", "image/jpeg,image/png,image/bmp,image/gif,image/webp,video/mp4,video/ogg,video/mpeg,video/webm");
        loader.send();
    }

    // TODO: Info modal (maybe have a separate root for it? should only poll info when opened)
    // TODO: bind various buttons to react actions
    // TODO: Keyboard controls
    render() {
        return (<div>
            <LargeImageMngr sel={this.state.listing.length ? "/" + this.state.curFldr + "/" + this.state.listing[this.state.lsind] : "flight.jpg"} />
            <SmallImageMngr sel={this.state.listing.length ? "/" + this.state.curFldr + "/" + this.state.listing[this.state.lsind] : "flight.jpg"} />
            <FolderMenuMngr folders={this.state.folders} onClick={(i) => this.handleFldrMenuClick(i)} />
        </div>
        );
    }
}

const root = ReactDOM.createRoot(document.getElementById("reactRoot"));
root.render(<GodObject />);
