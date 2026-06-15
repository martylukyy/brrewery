dnl AM_PATH_CPPUNIT(MINIMUM-VERSION, [ACTION-IF-FOUND [, ACTION-IF-NOT-FOUND]])
dnl
dnl Compatibility shim for rtorrent 0.9.6 / libtorrent 0.13.6, whose configure.ac
dnl calls the classic AM_PATH_CPPUNIT macro. That macro shipped in cppunit.m4 via
dnl cppunit-config, which modern distributions (e.g. Debian trixie's
dnl libcppunit-dev) no longer install. cppunit still provides a pkg-config file,
dnl so define the macro in terms of PKG_CHECK_MODULES and never abort: cppunit is
dnl only used by the (unbuilt-by-brrewery) test suite.
AC_DEFUN([AM_PATH_CPPUNIT],
[
  PKG_CHECK_MODULES([CPPUNIT], [cppunit >= $1],
    [m4_default([$2], [:])],
    [m4_default([$3], [:])])
  AC_SUBST([CPPUNIT_CFLAGS])
  AC_SUBST([CPPUNIT_LIBS])
])
