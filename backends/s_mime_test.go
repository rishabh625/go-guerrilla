package backends

import (
	"bytes"
	"fmt"
	"testing"
)

var p *parser

func init() {
	p = newMimeParser()
}
func TestInject(t *testing.T) {
	var b bytes.Buffer

	// it should read from both slices
	// as if it's a continuous stream
	p.inject([]byte("abcd"), []byte("efgh"), []byte("ijkl"))
	for i := 0; i < 12; i++ {
		b.WriteByte(p.ch)
		p.next()
		if p.ch == 0 {
			break
		}
	}
	if b.String() != "abcdefghijkl" {
		t.Error("expecting abcdefghijkl, got:", b.String())
	}
}
func TestMimeType(t *testing.T) {

	if isTokenSpecial['-'] {
		t.Error("- should not be in the set")
	}

	p.inject([]byte("text/plain; charset=us-ascii"))
	str, err := p.mimeType()
	if err != nil {
		t.Error(err)
	}
	if str != "text" {
		t.Error("mime type should be: text")
	}

}

func TestMimeContentType(t *testing.T) {
	go func() {
		<-p.consumed
		p.gotNewSlice <- false
	}()
	p.inject([]byte("text/plain; charset=us-ascii"))
	contentType, err := p.contentType()
	if err != nil {
		t.Error(err)
	}
	if contentType.subType != "plain" {
		t.Error("contentType.subType expecting 'plain', got:", contentType.subType)
	}

	if contentType.superType != "text" {
		t.Error("contentType.subType expecting 'text', got:", contentType.superType)
	}
}

func TestEmailHeader(t *testing.T) {
	in := `From: Al Gore <vice-president@whitehouse.gov>
To: White House Transportation Coordinator <transport@whitehouse.gov>
Subject: [Fwd: Map of Argentina with Description]
MIME-Version: 1.0
DKIM-Signature: v=1; a=rsa-sha256; c=relaxed; s=ncr424; d=reliancegeneral.co.in;
 h=List-Unsubscribe:MIME-Version:From:To:Reply-To:Date:Subject:Content-Type:Content-Transfer-Encoding:Message-ID; i=prospects@prospects.reliancegeneral.co.in;
 bh=F4UQPGEkpmh54C7v3DL8mm2db1QhZU4gRHR1jDqffG8=;
 b=MVltcq6/I9b218a370fuNFLNinR9zQcdBSmzttFkZ7TvV2mOsGrzrwORT8PKYq4KNJNOLBahswXf
   GwaMjDKT/5TXzegdX/L3f/X4bMAEO1einn+nUkVGLK4zVQus+KGqm4oP7uVXjqp70PWXScyWWkbT
   1PGUwRfPd/HTJG5IUqs=
Content-Type: multipart/mixed;
 boundary="D7F------------D7FD5A0B8AB9C65CCDBFA872"

This is a multi-part message in MIME format.
--D7F------------D7FD5A0B8AB9C65CCDBFA872
Content-Type: text/plain; charset=us-ascii
Content-Transfer-Encoding: 7bit

Fred,

Fire up Air Force One!  We\'re going South!

Thanks,
Al
--D7F------------D7FD5A0B8AB9C65CCDBFA872
This
`
	p.inject([]byte(in))
	h := NewMimeHeader()
	err := p.header(h)
	if err != nil {
		t.Error(err)
	}
	if _, err := p.boundary(h.contentBoundary); err != nil {
		t.Error(err)
	} else {
		//_ = part
		//p.addPart(part)

		//nextPart := NewMimeHeader()
		//err = p.body(part)
		//if err != nil {
		//	t.Error(err)
		//}
	}
}

func TestBoundary(t *testing.T) {
	var err error
	part := NewMimeHeader()
	part.contentBoundary = "-wololo-"

	// in the middle of the string
	p.inject([]byte("The quick brown fo-wololo-x jumped over the lazy dog"))

	_, err = p.boundary(part.contentBoundary)
	if err != nil {
		t.Error(err)
	}

	//for c := p.next(); c != 0; c= p.next() {} // drain

	p.inject([]byte("The quick brown fox jumped over the lazy dog-wololo-"))
	_, err = p.boundary(part.contentBoundary)
	if err != nil {
		t.Error(err)
	}

	for c := p.next(); c != 0; c = p.next() {
	} // drain

	// boundary is split over multiple slices
	p.inject(
		[]byte("The quick brown fox jumped ov-wolo"),
		[]byte("lo-er the lazy dog"))
	_, err = p.boundary(part.contentBoundary)
	if err != nil {
		t.Error(err)
	}
	for c := p.next(); c != 0; c = p.next() {
	} // drain
	// the boundary with an additional buffer in between
	p.inject([]byte("The quick brown fox jumped over the lazy dog"),
		[]byte("this is the middle"),
		[]byte("and thats the end-wololo-"))

	_, err = p.boundary(part.contentBoundary)
	if err != nil {
		t.Error(err)
	}

}

func TestMimeContentQuotedParams(t *testing.T) {

	// quoted
	p.inject([]byte("text/plain; charset=\"us-ascii\""))
	contentType, err := p.contentType()
	if err != nil {
		t.Error(err)
	}

	// with whitespace & tab
	p.inject([]byte("text/plain; charset=\"us-ascii\"  \tboundary=\"D7F------------D7FD5A0B8AB9C65CCDBFA872\""))
	contentType, err = p.contentType()
	if err != nil {
		t.Error(err)
	}

	// with comment (ignored)
	p.inject([]byte("text/plain; charset=\"us-ascii\" (a comment) \tboundary=\"D7F------------D7FD5A0B8AB9C65CCDBFA872\""))
	contentType, err = p.contentType()

	if contentType.subType != "plain" {
		t.Error("contentType.subType expecting 'plain', got:", contentType.subType)
	}

	if contentType.superType != "text" {
		t.Error("contentType.subType expecting 'text', got:", contentType.superType)
	}

	if len(contentType.parameters) != 2 {
		t.Error("expecting 2 elements in parameters")
	} else {
		if _, ok := contentType.parameters["charset"]; !ok {
			t.Error("charset parameter not present")
		}
		if b, ok := contentType.parameters["boundary"]; !ok {
			t.Error("charset parameter not present")
		} else {
			if b != "D7F------------D7FD5A0B8AB9C65CCDBFA872" {
				t.Error("boundary should be: D7F------------D7FD5A0B8AB9C65CCDBFA872")
			}
		}
	}

}

func msg() (err error) {
	main := NewMimeHeader()
	err = p.header(main)
	if err != nil {
		return err
	}
	p.addPart(main, "1")

	if main.contentBoundary != "" {
		// it's a message with mime parts

		if end, bErr := p.boundary(main.contentBoundary); bErr != nil {
			return bErr
		} else if end {
			return
		}

		if err = p.mimeMsg("", "1"); err != nil {
			return err
		}
	} else {
		// only contains one part (the body)
		if err := p.body(main); err != nil {
			return err
		}
	}
	p.endBody(main)

	return
}

var email = `From:  Al Gore <vice-president@whitehouse.gov>
To:  White House Transportation Coordinator <transport@whitehouse.gov>
Subject: [Fwd: Map of Argentina with Description]
MIME-Version: 1.0
DKIM-Signature: v=1; a=rsa-sha256; c=relaxed; s=ncr424; d=reliancegeneral.co.in;
 h=List-Unsubscribe:MIME-Version:From:To:Reply-To:Date:Subject:Content-Type:Content-Transfer-Encoding:Message-ID; i=prospects@prospects.reliancegeneral.co.in;
 bh=F4UQPGEkpmh54C7v3DL8mm2db1QhZU4gRHR1jDqffG8=;
 b=MVltcq6/I9b218a370fuNFLNinR9zQcdBSmzttFkZ7TvV2mOsGrzrwORT8PKYq4KNJNOLBahswXf
   GwaMjDKT/5TXzegdX/L3f/X4bMAEO1einn+nUkVGLK4zVQus+KGqm4oP7uVXjqp70PWXScyWWkbT
   1PGUwRfPd/HTJG5IUqs=
Content-Type: multipart/mixed;
 boundary="D7F------------D7FD5A0B8AB9C65CCDBFA872"

This is a multi-part message in MIME format.
--D7F------------D7FD5A0B8AB9C65CCDBFA872
Content-Type: text/plain; charset=us-ascii
Content-Transfer-Encoding: 7bit

Fred,

Fire up Air Force One!  We\'re going South!

Thanks,
Al
--D7F------------D7FD5A0B8AB9C65CCDBFA872
Content-Type: message/rfc822
Content-Transfer-Encoding: 7bit
Content-Disposition: inline

Return-Path: <president@whitehouse.gov>
Received: from mailhost.whitehouse.gov ([192.168.51.200])
 by heartbeat.whitehouse.gov (8.8.8/8.8.8) with ESMTP id SAA22453
 for <vice-president@heartbeat.whitehouse.gov>;
 Mon, 13 Aug 1998 l8:14:23 +1000
Received: from the_big_box.whitehouse.gov ([192.168.51.50])
 by mailhost.whitehouse.gov (8.8.8/8.8.7) with ESMTP id RAA20366
 for vice-president@whitehouse.gov; Mon, 13 Aug 1998 17:42:41 +1000
 Date: Mon, 13 Aug 1998 17:42:41 +1000
Message-Id: <199804130742.RAA20366@mai1host.whitehouse.gov>
From: Bill Clinton <president@whitehouse.gov>
To: A1 (The Enforcer) Gore <vice-president@whitehouse.gov>
Subject:  Map of Argentina with Description
MIME-Version: 1.0
Content-Type: multipart/mixed;
 boundary="DC8------------DC8638F443D87A7F0726DEF7"

This is a multi-part message in MIME format.
--DC8------------DC8638F443D87A7F0726DEF7
Content-Type: text/plain; charset=us-ascii
Content-Transfer-Encoding: 7bit

Hi A1,

I finally figured out this MIME thing.  Pretty cool.  I\'ll send you
some sax music in .au files next week!

Anyway, the attached image is really too small to get a good look at
Argentina.  Try this for a much better map:

http://www.1one1yp1anet.com/dest/sam/graphics/map-arg.htm

Then again, shouldn\'t the CIA have something like that?

Bill
--DC8------------DC8638F443D87A7F0726DEF7
Content-Type: image/gif; name="map_of_Argentina.gif"
Content-Transfer-Encoding: base64
Content-Disposition: in1ine; fi1ename="map_of_Argentina.gif"

R01GOD1hJQA1AKIAAP/////78P/omn19fQAAAAAAAAAAAAAAACwAAAAAJQA1AAAD7Qi63P5w
wEmjBCLrnQnhYCgM1wh+pkgqqeC9XrutmBm7hAK3tP31gFcAiFKVQrGFR6kscnonTe7FAAad
GugmRu3CmiBt57fsVq3Y0VFKnpYdxPC6M7Ze4crnnHum4oN6LFJ1bn5NXTN7OF5fQkN5WYow
BEN2dkGQGWJtSzqGTICJgnQuTJN/WJsojad9qXMuhIWdjXKjY4tenjo6tjVssk2gaWq3uGNX
U6ZGxseyk8SasGw3J9GRzdTQky1iHNvcPNNI4TLeKdfMvy0vMqLrItvuxfDW8ubjueDtJufz
7itICBxISKDBgwgTKjyYAAA7
--DC8------------DC8638F443D87A7F0726DEF7--

--D7F------------D7FD5A0B8AB9C65CCDBFA872--

`

func TestNestedEmail(t *testing.T) {
	p.inject([]byte(email))

	if err := p.mime("", "1"); err != nil {
		t.Error(err)
	}
	for part := range p.parts {
		fmt.Println(p.parts[part].part, " ", p.parts[part].contentType)
	}

}
