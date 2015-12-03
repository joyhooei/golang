#!/usr/bin/python

import time

from jpush import JPushClient


sendno = int(time.time())
app_key = 'c2c189e94edc71d5bc73ef15'
master_secret = '5673c63c7b8071c8647b394f'

jpush_client = JPushClient(master_secret)




from os import curdir,sep
from BaseHTTPServer import BaseHTTPRequestHandler,HTTPServer
import urlparse


class MyHandler(BaseHTTPRequestHandler):
	def do_GET(self):
		try:
			parsed = urlparse.urlparse(self.path)
			params = urlparse.parse_qs(parsed.query,0,1)
			self.send_response(200)
			self.end_headers()
			if params['type'][0] == 'notify' :
				if params['by'][0] == 'tag' :
					jpush_client.send_notification_by_alias(params['tag'][0], app_key, sendno, '',params['title'][0],params['content'][0], params['platform'][0])
				elif params['by'][0] == 'alias':
					jpush_client.send_notification_by_alias(params['alias'][0], app_key, sendno, '',params['title'][0],params['content'][0], params['platform'][0])
			elif params['type'][0] == 'msg' :
				if params['by'][0] == 'tag' :
					jpush_client.send_custom_msg_by_tag(params['tag'][0], app_key, sendno, '',params['title'][0],params['content'][0], params['platform'][0])
				elif params['by'][0] == 'alias':
					jpush_client.send_custom_msg_by_alias(params['alias'][0], app_key, sendno, '',params['title'][0],params['content'][0], params['platform'][0])
			self.wfile.write("{'status':'ok'}")
		except IOError:
			self.send_error(404, 'File Not Found: %s' % self.path)
		except Exception, e:
			print e,e.read()
			self.send_error(200,e)


try:
	server = HTTPServer(('localhost',8010),MyHandler)
	print 'welcome to the ,machine...',
	print 'Press ^C once or twice to quit'
	server.serve_forever()
except KeyboardInterrupt:
	print '^C received,shutting down server'
	server.socket.close()

if __name__=='__main__':
	main()
