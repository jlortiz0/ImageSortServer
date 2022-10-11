function FolderMenu(props) {
    return props.folders.map((x, i) => {
        return (
            <li className="w3-hover-dark-gray" onClick={() => props.onClick(i)} key={x}>{x}</li>
        );
    })
}

// TODO: Flag consts and proper empty handling, get folder list
class GodObject extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            folders: ["1", "2", "3"],
            curFldr: "",
            listing: [],
            flags: 0,
            lsind: 0,
        }
    }

    handleFldrMenuClick(i) {
        if (this.state.folders[i] != this.curFldr) {
            this.populateFldrList(this.state.folders[i]);
            this.setState({
                curFldr: this.state.folders[i],
                flags: 1,
            });
            document.getElementById('sidebar').style.display = 'none';
        }
    }

    populateFldrList(fldr) {
        const loader = new XMLHttpRequest();
        loader.onload = function () {
            if (loader.status != 200) {
                if (loader.responseText.length != 0) {
                    // TODO: Error modal
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
        }
        loader.open("GET", "/api/1/list/" + fldr);
        loader.send();
    }

    render() {
        return ReactDOM.createPortal(<FolderMenu folders={this.state.folders} onClick={(i) => this.handleFldrMenuClick(i)} />,
            document.getElementById("folderMenuMountPoint"));
    }
}

const root = ReactDOM.createRoot(document.getElementById("reactRoot"));
root.render(<GodObject />);
