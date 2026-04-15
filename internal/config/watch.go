package config

import "github.com/fsnotify/fsnotify"

type Watcher struct {
	fs      *fsnotify.Watcher
	path    string
	changes chan *Config
	errors  chan error
	done    chan struct{}
}

func Watch(path string) (*Watcher, error) {
	fs, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := fs.Add(path); err != nil {
		fs.Close()
		return nil, err
	}
	w := &Watcher{
		fs:      fs,
		path:    path,
		changes: make(chan *Config, 1),
		errors:  make(chan error, 1),
		done:    make(chan struct{}),
	}
	go w.loop()
	return w, nil
}

func (w *Watcher) Changes() <-chan *Config { return w.changes }
func (w *Watcher) Errors() <-chan error    { return w.errors }

func (w *Watcher) Close() {
	close(w.done)
	w.fs.Close()
}

func (w *Watcher) loop() {
	for {
		select {
		case <-w.done:
			return
		case ev, ok := <-w.fs.Events:
			if !ok {
				return
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
				cfg, err := Load(w.path)
				if err != nil {
					select {
					case w.errors <- err:
					default:
					}
					continue
				}
				select {
				case w.changes <- cfg:
				default:
				}
			}
		case err, ok := <-w.fs.Errors:
			if !ok {
				return
			}
			select {
			case w.errors <- err:
			default:
			}
		}
	}
}
