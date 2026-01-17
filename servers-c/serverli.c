#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <fcntl.h>
#include <errno.h>
#include <sys/socket.h>
#include <sys/epoll.h>
#include <netinet/in.h>

#define PORT 11
#define MAX_EVENTS 1024
#define BUFFER_SIZE 8192

typedef struct {
    int fd;
    char *data;
    size_t len;

    int headers_done;
    size_t content_length;
    size_t total_needed;
} client_t;

int set_nonblocking(int fd) {
    return fcntl(fd, F_SETFL, fcntl(fd, F_GETFL) | O_NONBLOCK);
}

size_t parse_content_length(const char *h) {
    const char *p = strcasestr(h, "Content-Length:");
    if (!p) return 0;
    return strtoul(p + 15, NULL, 10);
}

void send_response(client_t *c) {
    char *first_line_end = strstr(c->data, "\r\n");
    if (!first_line_end) return;

    size_t rest = c->len - (first_line_end + 2 - c->data);

    char *resp = malloc(c->len + 32);

    int n = snprintf(resp, 32, "HTTP/1.1 200 OK\r\n");
    memcpy(resp + n, first_line_end + 2, rest);

    write(c->fd, resp, n + rest);
    free(resp);
}

int main() {
    int server = socket(AF_INET, SOCK_STREAM, 0);

    int opt = 1;
    setsockopt(server, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));
    set_nonblocking(server);

    struct sockaddr_in addr = {
        .sin_family = AF_INET,
        .sin_port = htons(PORT),
        .sin_addr.s_addr = INADDR_ANY
    };

    bind(server, (struct sockaddr *)&addr, sizeof(addr));
    listen(server, 1024);

    int epfd = epoll_create1(0);

    struct epoll_event ev = {
        .events = EPOLLIN,
        .data.fd = server
    };
    epoll_ctl(epfd, EPOLL_CTL_ADD, server, &ev);

    struct epoll_event events[MAX_EVENTS];

    while (1) {
        int n = epoll_wait(epfd, events, MAX_EVENTS, -1);

        for (int i = 0; i < n; i++) {

            if (events[i].data.fd == server) {
                int client = accept(server, NULL, NULL);
                set_nonblocking(client);

                client_t *c = calloc(1, sizeof(client_t));
                c->fd = client;

                struct epoll_event ce = {
                    .events = EPOLLIN | EPOLLET,
                    .data.ptr = c
                };
                epoll_ctl(epfd, EPOLL_CTL_ADD, client, &ce);
                continue;
            }

            client_t *c = events[i].data.ptr;
            char buf[BUFFER_SIZE];

            ssize_t r;
            while ((r = read(c->fd, buf, sizeof(buf))) > 0) {

                c->data = realloc(c->data, c->len + r);
                memcpy(c->data + c->len, buf, r);
                c->len += r;

                if (!c->headers_done) {
                    char *h_end = strstr(c->data, "\r\n\r\n");
                    if (h_end) {
                        c->headers_done = 1;
                        c->content_length = parse_content_length(c->data);
                        size_t header_size = (h_end + 4) - c->data;
                        c->total_needed = header_size + c->content_length;
                    }
                }

                if (c->headers_done && c->len >= c->total_needed) {
                    send_response(c);

                    free(c->data);
                    c->data = NULL;
                    c->len = 0;
                    c->headers_done = 0;
                    c->content_length = 0;
                    c->total_needed = 0;
                }
            }

            if (r == 0 || (r < 0 && errno != EAGAIN)) {
                epoll_ctl(epfd, EPOLL_CTL_DEL, c->fd, NULL);
                close(c->fd);
                free(c->data);
                free(c);
            }
        }
    }
}