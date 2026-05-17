import asyncio
import os
import re
import sys
import logging

logging.basicConfig(level=logging.INFO, format='%(asctime)s [%(levelname)s] %(message)s')
logger = logging.getLogger('ws-terminal')

try:
    import websockets
except ImportError:
    logger.error('websockets not installed. Run: pip install websockets')
    raise

HOST = os.environ.get('WS_HOST', '0.0.0.0')
PORT = int(os.environ.get('WS_PORT', '5001'))
MAX_CONNECTIONS = 10
active_connections = set()

USE_PTY = sys.platform != 'win32'

if USE_PTY:
    import signal
    import struct
    import fcntl
    import termios
    import pty


async def handle_pty(websocket):
    pid, fd = pty.fork()
    if pid == 0:
        os.environ.pop('WS_HOST', None)
        os.environ.pop('WS_PORT', None)
        os.execve('/bin/bash', ['/bin/bash', '-l'], os.environ)
        os._exit(1)

    fl = fcntl.fcntl(fd, fcntl.F_GETFL)
    fcntl.fcntl(fd, fcntl.F_SETFL, fl | os.O_NONBLOCK)
    _set_winsize(fd, 80, 24)

    killed = False

    def kill():
        nonlocal killed
        if killed:
            return
        killed = True
        try:
            os.close(fd)
        except OSError:
            pass
        try:
            os.kill(pid, signal.SIGTERM)
        except OSError:
            pass

    async def read_pty():
        try:
            while not killed:
                try:
                    data = os.read(fd, 4096)
                    if not data:
                        break
                    await websocket.send(data)
                except (BlockingIOError, OSError):
                    await asyncio.sleep(0.01)
        except websockets.exceptions.ConnectionClosed:
            pass
        finally:
            kill()

    async def write_pty():
        try:
            async for msg in websocket:
                if not isinstance(msg, str):
                    continue
                if msg.startswith('\x1b[RESIZE:'):
                    m = re.search(r'RESIZE:(\d+);(\d+)', msg)
                    if m:
                        _set_winsize(fd, int(m.group(1)), int(m.group(2)))
                else:
                    os.write(fd, msg.encode('utf-8'))
        except websockets.exceptions.ConnectionClosed:
            pass
        finally:
            kill()

    await asyncio.gather(read_pty(), write_pty())


async def handle_pipe(websocket):
    shell = os.environ.get('SHELL', '')
    if not shell or not os.path.exists(shell):
        shell = 'cmd.exe' if sys.platform == 'win32' else '/bin/bash'

    args = [shell]
    if sys.platform != 'win32':
        args.append('-l')

    process = await asyncio.create_subprocess_exec(
        *args,
        stdin=asyncio.subprocess.PIPE,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.STDOUT,
    )

    killed = False

    def kill():
        nonlocal killed
        if killed:
            return
        killed = True
        try:
            process.terminate()
        except:
            pass

    async def read_stdout():
        try:
            while not killed:
                data = await process.stdout.read(4096)
                if not data:
                    break
                await websocket.send(data.decode('utf-8', errors='replace'))
        except websockets.exceptions.ConnectionClosed:
            pass
        finally:
            kill()

    async def write_stdin():
        try:
            async for msg in websocket:
                if not isinstance(msg, str):
                    continue
                process.stdin.write(msg.encode('utf-8'))
                await process.stdin.drain()
        except websockets.exceptions.ConnectionClosed:
            pass
        finally:
            kill()

    await asyncio.gather(read_stdout(), write_stdin())


handle_terminal = handle_pty if USE_PTY else handle_pipe


def _set_winsize(fd, cols, rows):
    if not USE_PTY:
        return
    buf = struct.pack('HHHH', rows, cols, 0, 0)
    try:
        fcntl.ioctl(fd, termios.TIOCSWINSZ, buf)
    except OSError:
        pass


async def connection_handler(websocket):
    if len(active_connections) >= MAX_CONNECTIONS:
        await websocket.close(1013, 'too many connections')
        return

    active_connections.add(websocket)
    addr = websocket.remote_address
    logger.info(f'connect: {addr}  (active: {len(active_connections)})')

    try:
        await handle_terminal(websocket)
    except Exception as e:
        logger.error(f'error: {e}')
    finally:
        active_connections.discard(websocket)
        logger.info(f'disconnect: {addr}  (active: {len(active_connections)})')


async def main():
    mode = 'PTY' if USE_PTY else 'pipe'
    logger.info(f'starting on ws://{HOST}:{PORT}  (mode: {mode})')
    async with websockets.serve(connection_handler, HOST, PORT):
        await asyncio.Future()


if __name__ == '__main__':
    asyncio.run(main())
